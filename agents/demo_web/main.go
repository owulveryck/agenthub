package main

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//go:embed index.html
var staticFS embed.FS

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 1. WebSocket hub
	hub := NewHub(logger)
	go hub.Run()

	// 2. Start child processes (broker, cortex, echo_agent)
	pm := NewProcessManager(hub, logger)
	if err := pm.Start(ctx); err != nil {
		logger.Error("failed to start child processes", "error", err)
		os.Exit(1)
	}

	// 3. Connect gRPC chat client to broker
	chat, err := NewChatClient(hub, logger)
	if err != nil {
		logger.Error("failed to create chat client", "error", err)
		pm.Shutdown()
		os.Exit(1)
	}
	if err := chat.Start(ctx); err != nil {
		logger.Error("failed to start chat client", "error", err)
		pm.Shutdown()
		os.Exit(1)
	}

	// Wire browser chat messages to the gRPC client
	hub.onChatMessage = func(text string) {
		go chat.SendMessage(ctx, text)
	}

	// 4. HTTP server
	addr := os.Getenv("DEMO_WEB_ADDR")
	if addr == "" {
		addr = ":8090"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.ServeWS)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := staticFS.ReadFile("index.html")
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		logger.Info("HTTP server starting", "addr", addr)
		fmt.Fprintf(os.Stderr, "\n  Open http://localhost%s in your browser\n\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	logger.Info("shutdown signal received")

	// Graceful shutdown in reverse order
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Stop HTTP server
	srv.Shutdown(shutdownCtx)

	// Stop chat client
	cancel()
	chat.Shutdown(shutdownCtx)

	// Stop child processes
	pm.Shutdown()

	logger.Info("shutdown complete")
}

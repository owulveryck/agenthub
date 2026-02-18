package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	mux.HandleFunc("/upload", handleUpload(logger))
	mux.HandleFunc("/agent/mp3/start", handleAgentStart(pm, ctx))
	mux.HandleFunc("/agent/mp3/stop", handleAgentStop(pm))
	mux.HandleFunc("/agent/mp3/status", handleAgentStatus(pm))
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

const maxUploadSize = 50 << 20 // 50 MB

func handleUpload(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

		file, header, err := r.FormFile("file")
		if err != nil {
			logger.Error("upload: failed to read form file", "error", err)
			http.Error(w, "failed to read uploaded file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		ext := ".mp3"
		if header != nil {
			if e := strings.ToLower(filepath.Ext(header.Filename)); e == ".m4a" {
				ext = ".m4a"
			}
		}

		tmp, err := os.CreateTemp("", "agenthub-*"+ext)
		if err != nil {
			logger.Error("upload: failed to create temp file", "error", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		defer tmp.Close()

		if _, err := io.Copy(tmp, file); err != nil {
			logger.Error("upload: failed to write temp file", "error", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		logger.Info("file uploaded", "path", tmp.Name())

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"path": tmp.Name()})
	}
}

func handleAgentStart(pm *ProcessManager, ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := pm.StartAgent(ctx, Mp3AgentSpec); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "started"})
	}
}

func handleAgentStop(pm *ProcessManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := pm.StopAgent("mp3_agent"); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
	}
}

func handleAgentStatus(pm *ProcessManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"running": pm.IsRunning("mp3_agent")})
	}
}

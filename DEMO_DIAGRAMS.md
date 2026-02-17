# Diagrammes Conceptuels AgentHub & Agent2Agent

Ce document présente la solution AgentHub et le protocole Agent2Agent à travers des diagrammes conceptuels qui permettent de comprendre l'architecture, les flux et les patterns sans utiliser de diagrammes de séquence.

## Table des Matières

1. [Architecture Globale: Agent2Agent avec AgentHub](#1-architecture-globale-agent2agent-avec-agenthub)
2. [Les Acteurs du Système](#2-les-acteurs-du-système)
3. [Le Workflow de Communication](#3-le-workflow-de-communication)
4. [Cortex: L'Orchestrateur Intelligent](#4-cortex-lorchest rateur-intelligent)
5. [Structure d'un Message A2A](#5-structure-dun-message-a2a)
6. [Les Avantages de la Solution](#6-les-avantages-de-la-solution)
7. [Cas d'Usage Typiques](#7-cas-dusage-typiques)
8. [De la Complexité à la Simplicité avec SubAgent](#8-de-la-complexité-à-la-simplicité-avec-subagent)
9. [Vision Globale de la Démo](#9-vision-globale-de-la-démo)

---

## 1. Architecture Globale: Agent2Agent avec AgentHub

```mermaid
graph TB
    subgraph "Protocole Agent2Agent (A2A)"
        A2A_SPEC["🔵 Protocole A2A<br/>• Structure des Messages<br/>• Format des Tasks<br/>• États & Priorités<br/>• Communication Patterns"]
    end

    subgraph "AgentHub - Implémentation"
        BROKER["⚡ Event Bus Broker<br/>• Routage centralisé<br/>• Pub/Sub Architecture<br/>• gRPC Services"]

        AGENTS["🤖 Agents<br/>• Publisher Agents<br/>• Subscriber Agents<br/>• Cortex Orchestrator"]

        SUBAGENT["📦 SubAgent Library<br/>• Abstraction simplifiée<br/>• Auto-configuration<br/>• Observabilité automatique"]
    end

    subgraph "Observabilité"
        OBS["📊 Stack Observabilité<br/>• OpenTelemetry<br/>• Jaeger (traces)<br/>• Prometheus (métriques)<br/>• Grafana (dashboards)"]
    end

    A2A_SPEC -.->|"définit"| BROKER
    A2A_SPEC -.->|"définit"| AGENTS
    SUBAGENT -->|"simplifie"| AGENTS
    BROKER <-->|"événements"| AGENTS
    AGENTS -->|"métriques/traces"| OBS
    BROKER -->|"métriques/traces"| OBS

    classDef protocol fill:#e1f5ff,stroke:#0288d1,stroke-width:2px
    classDef impl fill:#fff3e0,stroke:#f57c00,stroke-width:2px
    classDef obs fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px

    class A2A_SPEC protocol
    class BROKER,AGENTS,SUBAGENT impl
    class OBS obs
```

**Points clés:**
- **Agent2Agent (A2A)** est le protocole standardisé de communication
- **AgentHub** est l'implémentation du broker et de l'infrastructure
- **SubAgent Library** simplifie le développement d'agents
- **Observabilité complète** intégrée dès le départ

---

## 2. Les Acteurs du Système

```mermaid
graph LR
    subgraph "Types d'Agents"
        PUBLISHER["📤 Publisher Agent<br/>─────────────<br/>CRÉE les tasks<br/>SOUMET au broker<br/>REÇOIT les résultats<br/><br/>Exemple: Chat CLI"]

        SUBSCRIBER["📥 Subscriber Agent<br/>─────────────<br/>S'ABONNE aux tasks<br/>TRAITE les tasks<br/>PUBLIE les résultats<br/><br/>Exemple: Echo Agent"]

        ORCHESTRATOR["🎭 Orchestrator<br/>─────────────<br/>COORDONNE les agents<br/>ROUTE intelligemment<br/>MAINTIENT le contexte<br/><br/>Exemple: Cortex"]
    end

    subgraph "Communication Hub"
        BROKER_CENTER["⚡ Broker<br/>─────────────<br/>ROUTE les messages<br/>GÈRE les souscriptions<br/>ASSURE la livraison"]
    end

    PUBLISHER -->|"publie tasks"| BROKER_CENTER
    BROKER_CENTER -->|"distribue tasks"| SUBSCRIBER
    SUBSCRIBER -->|"publie résultats"| BROKER_CENTER
    BROKER_CENTER -->|"retourne résultats"| PUBLISHER
    ORCHESTRATOR <-->|"coordonne via"| BROKER_CENTER

    classDef agent fill:#e8f5e9,stroke:#388e3c,stroke-width:2px
    classDef broker fill:#fff9c4,stroke:#f57f17,stroke-width:3px

    class PUBLISHER,SUBSCRIBER,ORCHESTRATOR agent
    class BROKER_CENTER broker
```

**Rôles:**
- **Publisher**: Créateur de travail qui délègue
- **Subscriber**: Exécutant spécialisé qui traite
- **Orchestrator**: Coordinateur intelligent (Cortex)
- **Broker**: Hub de communication central

---

## 3. Le Workflow de Communication

```mermaid
graph TB
    START["👤 Utilisateur"]

    subgraph "Flux de Communication"
        SEND["1️⃣ ENVOI<br/>Message A2A créé<br/>• MessageId unique<br/>• ContextId (conversation)<br/>• Role: USER<br/>• Content structuré"]

        ROUTE["2️⃣ ROUTAGE<br/>Broker analyse<br/>• Direct → agent spécifique<br/>• Broadcast → tous agents<br/>• Topic-based → filtrage"]

        PROCESS["3️⃣ TRAITEMENT<br/>Agent exécute<br/>• Accepte/Rejette task<br/>• Traite de manière asynchrone<br/>• Génère artéfacts"]

        PROGRESS["4️⃣ SUIVI<br/>Updates en continu<br/>• SUBMITTED<br/>• WORKING (+ %)<br/>• COMPLETED/FAILED"]

        RESULT["5️⃣ RÉSULTAT<br/>Retour au demandeur<br/>• Artifact avec résultats<br/>• Metadata enrichie<br/>• Traçabilité complète"]
    end

    RECEIVE["👤 Utilisateur<br/>reçoit le résultat"]

    START --> SEND
    SEND --> ROUTE
    ROUTE --> PROCESS
    PROCESS --> PROGRESS
    PROGRESS --> RESULT
    RESULT --> RECEIVE

    PROGRESS -.->|"updates temps réel"| START

    classDef user fill:#e1bee7,stroke:#8e24aa,stroke-width:2px
    classDef step fill:#b3e5fc,stroke:#0277bd,stroke-width:2px

    class START,RECEIVE user
    class SEND,ROUTE,PROCESS,PROGRESS,RESULT step
```

**Étapes:**
1. **Envoi**: Création d'un message A2A structuré
2. **Routage**: Le broker détermine la destination
3. **Traitement**: L'agent exécute de façon autonome
4. **Suivi**: Updates de progression en temps réel
5. **Résultat**: Retour avec traçabilité complète

---

## 4. Cortex: L'Orchestrateur Intelligent

```mermaid
graph TB
    subgraph "Cortex - Le Cerveau du Système"
        direction TB

        LLM["🧠 LLM Decision Engine<br/>─────────────<br/>Analyse le contexte<br/>Détecte l'intention<br/>Choisit l'action"]

        STATE["💾 State Manager<br/>─────────────<br/>Historique conversations<br/>Tasks en cours<br/>Agents disponibles"]

        REGISTRY["📋 Agent Registry<br/>─────────────<br/>Découverte dynamique<br/>Capacités (Skills)<br/>Disponibilité"]
    end

    subgraph "Décisions Cortex"
        DIRECT["💬 Réponse Directe<br/>─────────────<br/>Requête simple<br/>Pas besoin d'agent<br/>Latence minimale"]

        DELEGATE["🎯 Délégation Agent<br/>─────────────<br/>Tâche spécialisée<br/>Choix de l'agent<br/>Suivi asynchrone"]

        MULTI["🔀 Orchestration Multi-Agents<br/>─────────────<br/>Tâche complexe<br/>Coordination pipeline<br/>Agrégation résultats"]
    end

    USER["👤 Utilisateur"]

    USER -->|"message"| LLM
    LLM <--> STATE
    LLM <--> REGISTRY

    LLM -->|"intent simple"| DIRECT
    LLM -->|"besoin skill"| DELEGATE
    LLM -->|"workflow complexe"| MULTI

    DIRECT -->|"réponse immédiate"| USER
    DELEGATE -->|"via agents"| USER
    MULTI -->|"résultat agrégé"| USER

    classDef cortex fill:#fff3e0,stroke:#f57c00,stroke-width:2px
    classDef decision fill:#e8f5e9,stroke:#388e3c,stroke-width:2px
    classDef user fill:#e1bee7,stroke:#8e24aa,stroke-width:2px

    class LLM,STATE,REGISTRY cortex
    class DIRECT,DELEGATE,MULTI decision
    class USER user
```

**Cortex** est le "cerveau" qui:
- Analyse l'intention de l'utilisateur via LLM
- Maintient le contexte conversationnel
- Découvre dynamiquement les agents disponibles
- Décide de répondre directement ou déléguer
- Orchestre plusieurs agents si nécessaire

---

## 5. Structure d'un Message A2A

```mermaid
graph TB
    subgraph "Message A2A - Format Standard"
        MSG["📧 Message"]

        META_ID["🔑 Identifiants<br/>• message_id<br/>• context_id<br/>• task_id"]

        META_ROLE["👤 Métadonnées<br/>• role (USER/AGENT)<br/>• timestamp<br/>• metadata"]

        CONTENT["📄 Contenu (Parts)<br/>• Text (texte simple)<br/>• Data (JSON structuré)<br/>• File (fichiers)"]
    end

    subgraph "Task A2A - Unité de Travail"
        TASK["⚙️ Task"]

        TASK_ID["🔑 Identifiants<br/>• task_id<br/>• context_id"]

        TASK_STATUS["📊 Status<br/>• state (SUBMITTED/WORKING/COMPLETED)<br/>• update message<br/>• timestamp"]

        TASK_RESULT["📦 Résultats<br/>• artifacts<br/>• history<br/>• metadata"]
    end

    subgraph "Routing - Instructions d'Acheminement"
        ROUTING["🚦 AgentEventMetadata"]

        ROUTE_AGENTS["🎯 Agents<br/>• from_agent_id<br/>• to_agent_id<br/>• subscriptions (topics)"]

        ROUTE_PRIO["⚡ Priorité<br/>• priority level<br/>• event_type"]
    end

    MSG --> META_ID
    MSG --> META_ROLE
    MSG --> CONTENT

    TASK --> TASK_ID
    TASK --> TASK_STATUS
    TASK --> TASK_RESULT

    ROUTING --> ROUTE_AGENTS
    ROUTING --> ROUTE_PRIO

    classDef message fill:#e3f2fd,stroke:#1976d2,stroke-width:2px
    classDef task fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    classDef routing fill:#fff3e0,stroke:#f57c00,stroke-width:2px

    class MSG,META_ID,META_ROLE,CONTENT message
    class TASK,TASK_ID,TASK_STATUS,TASK_RESULT task
    class ROUTING,ROUTE_AGENTS,ROUTE_PRIO routing
```

**Structure A2A:**
- **Message**: Unité de communication avec identifiants et rôle
- **Task**: Unité de travail avec état et résultats
- **Routing**: Instructions pour l'acheminement

---

## 6. Les Avantages de la Solution

```mermaid
mindmap
    root((AgentHub<br/>Agent2Agent))
        Architecture
            Event-Driven
                Asynchrone
                Non-bloquant
                Scalable
            Protocole A2A
                Interopérable
                Standardisé
                Structuré
            Hexagonale
                Modulaire
                Testable
                Évolutif
        Performance
            10,000+ tasks/sec
            Latence sub-ms
            Overhead minimal
                5% CPU
                50MB RAM
        Observabilité
            Tracing distribué
                OpenTelemetry
                Jaeger
            Métriques temps réel
                Prometheus
                Grafana
            Logs structurés
        Développement
            SubAgent Library
                75% moins de code
                Config déclarative
                Observabilité auto
            Skills-based
                Réutilisable
                Composable
                Découvrable
            LLM-powered
                Routing intelligent
                Auto-discovery
                Contexte-aware
```

---

## 7. Cas d'Usage Typiques

```mermaid
graph TB
    subgraph "Pattern 1: Request-Response Simple"
        RR1["Agent A<br/>Demande conversion CSV→JSON"]
        RR2["Agent B<br/>Convertit et retourne"]
        RR1 -->|"task"| RR2
        RR2 -->|"artifact"| RR1
    end

    subgraph "Pattern 2: Pipeline de Traitement"
        P1["Agent Extract<br/>Extraction données"]
        P2["Agent Transform<br/>Transformation"]
        P3["Agent Load<br/>Chargement"]
        P1 -->|"context_id partagé"| P2
        P2 -->|"context_id partagé"| P3
    end

    subgraph "Pattern 3: Orchestration Multi-Agents"
        O1["Cortex<br/>Coordonnateur"]
        O2["Agent Financier"]
        O3["Agent Marketing"]
        O4["Agent Technique"]
        O1 -->|"analyse financière"| O2
        O1 -->|"analyse marché"| O3
        O1 -->|"analyse technique"| O4
        O2 -->|"résultats"| O1
        O3 -->|"résultats"| O1
        O4 -->|"résultats"| O1
    end

    subgraph "Pattern 4: Broadcast Concurrent"
        B1["Orchestrateur<br/>Task à distribuer"]
        B2["Worker 1"]
        B3["Worker 2"]
        B4["Worker 3"]
        B1 -.->|"broadcast"| B2
        B1 -.->|"broadcast"| B3
        B1 -.->|"broadcast"| B4
    end

    classDef agent fill:#e8f5e9,stroke:#388e3c,stroke-width:2px
    classDef orchestrator fill:#fff3e0,stroke:#f57c00,stroke-width:2px

    class RR1,RR2,P1,P2,P3,O2,O3,O4,B2,B3,B4 agent
    class O1,B1 orchestrator
```

**Patterns disponibles:**
1. **Request-Response**: Communication simple 1-à-1
2. **Pipeline**: Chaînage d'agents avec contexte partagé
3. **Orchestration**: Coordination intelligente via Cortex
4. **Broadcast**: Distribution parallèle

---

## 8. De la Complexité à la Simplicité avec SubAgent

```mermaid
graph LR
    subgraph "Sans SubAgent Library - ~200 lignes"
        direction TB
        OLD1["⚙️ Setup gRPC<br/>45 lignes"]
        OLD2["📋 AgentCard Manual<br/>30 lignes"]
        OLD3["🔄 Subscription Logic<br/>60 lignes"]
        OLD4["📊 Observabilité Manual<br/>40 lignes"]
        OLD5["🔧 Lifecycle Management<br/>25 lignes"]
    end

    subgraph "Avec SubAgent Library - ~50 lignes"
        direction TB
        NEW1["⚡ Configuration<br/>10 lignes"]
        NEW2["🎯 Skills Registration<br/>5 lignes"]
        NEW3["💼 Business Logic<br/>30 lignes"]
        NEW4["▶️ Run<br/>2 lignes"]
        NEW5["✨ Tout le reste:<br/>AUTOMATIQUE!"]
    end

    BEFORE["🔴 Avant<br/>Complexe<br/>Boilerplate<br/>Erreurs"]
    AFTER["🟢 Après<br/>Simple<br/>Focus métier<br/>Fiable"]

    BEFORE --> OLD1
    OLD1 --> OLD2
    OLD2 --> OLD3
    OLD3 --> OLD4
    OLD4 --> OLD5

    AFTER --> NEW1
    NEW1 --> NEW2
    NEW2 --> NEW3
    NEW3 --> NEW4
    NEW4 --> NEW5

    classDef old fill:#ffebee,stroke:#c62828,stroke-width:2px
    classDef new fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px

    class OLD1,OLD2,OLD3,OLD4,OLD5,BEFORE old
    class NEW1,NEW2,NEW3,NEW4,NEW5,AFTER new
```

**Gain avec SubAgent Library:**
- **75% de code en moins** (200 → 50 lignes)
- **Configuration déclarative** au lieu d'impérative
- **Observabilité automatique** intégrée
- **Focus sur la logique métier** uniquement

---

## 9. Vision Globale de la Démo

```mermaid
graph TB
    subgraph "Démo AgentHub: Chat Interactif"
        USER["👤 Utilisateur<br/>via Chat CLI"]

        CLI["💬 Chat REPL Agent<br/>• Interface utilisateur<br/>• Envoi messages<br/>• Affichage réponses"]

        BROKER_DEMO["⚡ Event Bus Broker<br/>• Routage central<br/>• Pub/Sub<br/>• gRPC"]

        CORTEX_DEMO["🧠 Cortex<br/>• Orchestration<br/>• LLM decisions<br/>• State management"]

        ECHO["🔊 Echo Agent<br/>• Répète messages<br/>• Skill simple<br/>• Démo fonctionnement"]

        MONITORING["📊 Observabilité Stack<br/>• Jaeger: traces distribuées<br/>• Prometheus: métriques<br/>• Grafana: dashboards"]
    end

    USER <-->|"tape commandes"| CLI
    CLI <-->|"messages A2A"| BROKER_DEMO
    BROKER_DEMO <-->|"événements"| CORTEX_DEMO
    BROKER_DEMO <-->|"tasks"| ECHO

    CLI -.->|"traces"| MONITORING
    BROKER_DEMO -.->|"traces"| MONITORING
    CORTEX_DEMO -.->|"traces"| MONITORING
    ECHO -.->|"traces"| MONITORING

    classDef user fill:#e1bee7,stroke:#8e24aa,stroke-width:3px
    classDef agent fill:#e8f5e9,stroke:#388e3c,stroke-width:2px
    classDef infra fill:#fff3e0,stroke:#f57c00,stroke-width:2px
    classDef obs fill:#e3f2fd,stroke:#1976d2,stroke-width:2px

    class USER user
    class CLI,ECHO agent
    class BROKER_DEMO,CORTEX_DEMO infra
    class MONITORING obs
```

**Démo complète:**
- Utilisateur interagit via Chat CLI
- Messages routés par le Broker
- Cortex orchestre intelligemment
- Echo Agent démontre l'exécution
- Observabilité complète de bout en bout

---

## Points Clés à Retenir

### 🔵 Agent2Agent (A2A)
- **Protocole standardisé** de communication entre agents autonomes
- **Messages structurés** avec identifiants, rôles, et contenu
- **Asynchrone par nature** pour opérations longues
- **Interopérable** entre différentes implémentations

### ⚡ AgentHub Broker
- **Event Bus centralisé** pour routage de messages
- **Pub/Sub architecture** scalable et performante
- **Routage flexible**: direct, broadcast, topic-based
- **Performance**: 10,000+ tasks/sec, latence sub-ms

### 🧠 Cortex Orchestrator
- **Cerveau du système** avec décisions LLM
- **Gestion d'état conversationnel**
- **Découverte dynamique** des agents
- **Non-bloquant**: utilisateur peut continuer à interagir

### 📦 SubAgent Library
- **Abstraction simplifiée**: 75% de code en moins
- **Configuration déclarative**
- **Observabilité automatique** (traces, logs, métriques)
- **Skills-based programming** pour modularité

### 📊 Observabilité
- **OpenTelemetry** pour tracing distribué
- **Traces end-to-end** avec corrélation
- **Métriques temps réel** via Prometheus
- **Dashboards** Grafana pour visualisation

---

## Qu'est-ce qu'Agent2Agent fait concrètement?

```mermaid
graph TB
    PROBLEM["❌ Problème Traditionnel<br/>━━━━━━━━━━━━━━━━<br/>• Services couplés<br/>• Communication synchrone<br/>• Pas de visibilité<br/>• Difficile à scaler<br/>• Rigide"]

    A2A["✅ Solution Agent2Agent<br/>━━━━━━━━━━━━━━━━"]

    subgraph "Ce que A2A apporte"
        DECOUPLE["🔓 Découplage<br/>Agents autonomes<br/>Communication async<br/>Event-driven"]

        STANDARD["📋 Standardisation<br/>Format messages uniforme<br/>Interopérabilité<br/>Protocol-first"]

        OBSERVE["👁️ Observabilité<br/>Tracing distribué<br/>Corrélation tasks<br/>Métriques end-to-end"]

        SCALE["📈 Scalabilité<br/>10,000+ tasks/sec<br/>Horizontal scaling<br/>Performance garantie"]

        FLEX["🎯 Flexibilité<br/>Hot-plug agents<br/>Skills dynamiques<br/>LLM routing"]
    end

    BENEFITS["🎁 Bénéfices<br/>━━━━━━━━━━━━━━━━<br/>✓ Développement rapide<br/>✓ Maintenance simple<br/>✓ Extensibilité facile<br/>✓ Production-ready<br/>✓ Observabilité complète"]

    PROBLEM --> A2A
    A2A --> DECOUPLE
    A2A --> STANDARD
    A2A --> OBSERVE
    A2A --> SCALE
    A2A --> FLEX

    DECOUPLE --> BENEFITS
    STANDARD --> BENEFITS
    OBSERVE --> BENEFITS
    SCALE --> BENEFITS
    FLEX --> BENEFITS

    classDef problem fill:#ffebee,stroke:#c62828,stroke-width:2px
    classDef solution fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px
    classDef benefit fill:#e3f2fd,stroke:#1976d2,stroke-width:2px

    class PROBLEM problem
    class A2A,DECOUPLE,STANDARD,OBSERVE,SCALE,FLEX solution
    class BENEFITS benefit
```

**En résumé, Agent2Agent:**
1. **Définit un protocole** de communication standardisé pour agents
2. **Permet l'autonomie** - chaque agent décide quoi faire
3. **Assure l'asynchronisme** - pas de blocage
4. **Fournit la traçabilité** - chaque message/task est tracé
5. **Garantit la performance** - 10,000+ tasks/sec
6. **Simplifie le développement** - via SubAgent Library

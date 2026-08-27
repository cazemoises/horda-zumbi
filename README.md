# Horda Zumbi

Jogo cooperativo de sobrevivência a hordas de zumbis, multiplayer via rede
(cada jogador em seu próprio navegador), até 3 jogadores por sala. Sem
persistência: todo o estado (salas, jogadores, zumbis) vive em memória no
processo do backend e é perdido a cada reinício.

**Status:** ainda não está deployado em lugar nenhum. Este README cobre
apenas execução local.

## Estrutura

```
horda-zumbi/
├── backend/    # Go, arquitetura hexagonal (domain/ports/adapters)
├── frontend/   # Vite + React + TS, renderização em canvas HTML5
└── docker-compose.yml
```

## Rodando local via Docker Compose

```
docker compose up --build
```

- Backend: http://localhost:8080 (`GET /health`, WebSocket em `/ws/rooms/{code}`)
- Frontend: http://localhost:8081 (proxya `/ws` e `/health` para o backend)

Portas configuráveis via `BACKEND_PORT` e `FRONTEND_PORT`.

## Rodando local sem Docker

Backend:

```
cd backend
go run ./cmd/server
```

Escuta em `:8080` por padrão (configurável via variável de ambiente `PORT`).

Frontend:

```
cd frontend
npm install
npm run dev
```

O servidor de desenvolvimento do Vite sobe em `http://localhost:5173`.

## Testando multiplayer

Como o jogo é multiplayer via rede (sem sala compartilhada na mesma tela),
para testar localmente com "vários jogadores":

1. Suba o backend (Docker ou `go run`).
2. Abra várias abas (ou janelas anônimas) do navegador apontando para o
   frontend.
3. Em cada aba, conecte ao mesmo código de sala (ex.: `/ws/rooms/abc123`).
   Cada aba representa um jogador diferente; mensagens enviadas por uma aba
   são retransmitidas às outras conectadas à mesma sala.
4. Uma sala aceita no máximo 3 conexões simultâneas; a quarta é recusada.

No estágio atual, o backend só faz a fundação de rede (relay de mensagens
entre clientes da mesma sala) - ainda não há lógica de jogo (movimento,
combate, ondas de zumbis) implementada.

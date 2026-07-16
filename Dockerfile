# ── Stage 1: build the frontend ────────────────────────────────────────────
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# ── Stage 2: build the Go binary ───────────────────────────────────────────
FROM golang:1.25-alpine AS backend
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -o kosh .

# ── Stage 3: slim runtime image ────────────────────────────────────────────
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=backend /app/backend/kosh ./kosh
COPY --from=backend /app/backend/prompts ./prompts
COPY --from=backend /app/backend/dev_data.json ./dev_data.json
COPY --from=frontend /app/frontend/dist ./frontend/dist

ENV FRONTEND_DIST=/app/frontend/dist
ENV PROMPTS_DIR=/app/prompts
ENV PORT=8080
ENV CREDENTIALS_PATH=/app/credentials.json

EXPOSE 8080
# Most container platforms (Cloud Run included) have no built-in "secret file"
# upload — GOOGLE_CREDENTIALS_B64 (the service account JSON, base64-encoded)
# is decoded to disk at startup instead.
CMD ["/bin/sh", "-c", "if [ -n \"$GOOGLE_CREDENTIALS_B64\" ]; then echo \"$GOOGLE_CREDENTIALS_B64\" | base64 -d > \"$CREDENTIALS_PATH\"; fi; exec ./kosh"]

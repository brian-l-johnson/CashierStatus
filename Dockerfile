# syntax=docker/dockerfile:1

# --- Build stage ---
FROM golang:1.22 AS build
WORKDIR /src

# Cache module downloads as their own layer.
COPY go.mod go.sum ./
RUN go mod download

# Build a static, pure-Go binary (modernc sqlite => no cgo needed).
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cashier-status .

# Stage runtime assets alongside the binary so the whole tree can be copied
# into the final image with a single, ownership-controlled COPY.
RUN cp -r static templates /out/

# Pre-create the data directory so it can be copied in with nonroot ownership
# (distroless has no shell to mkdir/chown in the final stage).
RUN mkdir -p /data

# --- Final stage ---
# distroless "static" is the right base for a CGO_ENABLED=0 binary.
# The :nonroot variant runs as uid 65532 for least privilege.
FROM gcr.io/distroless/static-debian12:nonroot AS final
WORKDIR /app

# Copy the staged tree (binary + static + templates) owned by nonroot so the
# app can also create cashierstatus.db in this directory at runtime.
COPY --from=build --chown=nonroot:nonroot /out /app

# Persist the SQLite database on a writable, nonroot-owned volume so it
# survives container restarts and redeploys. Mount a host dir or named
# volume at /data to keep state outside the image.
COPY --from=build --chown=nonroot:nonroot /data /data
VOLUME ["/data"]
ENV DB_PATH=/data/cashierstatus.db

USER nonroot:nonroot
ENV GIN_MODE=release

# Gin's r.Run() listens on :8080 (honors $PORT if set).
EXPOSE 8080

ENTRYPOINT ["/app/cashier-status"]

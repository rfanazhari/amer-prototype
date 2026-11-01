FROM golang:1.26.10 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/amer-prototype ./cmd/app
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/amer-prototype /amer-prototype
COPY --from=build /out/migrate /migrate
COPY --from=build /src/migrations/pgsql /migrations/pgsql
WORKDIR /
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/amer-prototype"]

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=none
RUN CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
      -o /out/culprit ./cmd/culprit

# Culprit reads history through the git command line, so the runtime image
# carries git and nothing else.
FROM alpine:3.20
RUN apk add --no-cache git && adduser -D -u 10001 culprit
COPY --from=build /out/culprit /usr/local/bin/culprit
USER culprit
WORKDIR /home/culprit
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/culprit"]
CMD ["serve", "-repo", "/repo", "-listen", "0.0.0.0:8080"]

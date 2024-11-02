FROM node:22 AS frontend

WORKDIR /app

COPY front/package.json front/package-lock.json ./
RUN npm install

COPY front .
RUN npm run build

FROM golang:1.23 AS backend

WORKDIR /app

ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download -x
COPY . .
RUN go build -o app .

FROM gcr.io/distroless/static-debian12

WORKDIR /app
COPY --from=frontend /app/build ./build
COPY --from=backend /app/app ./app

CMD ["./app"]

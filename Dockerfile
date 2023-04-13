FROM node:lts-alpine as build-frontend
WORKDIR /app
COPY ./frontend/package*.json ./
RUN npm install
COPY ./frontend/ .
RUN npm run build

FROM golang:1.20-alpine AS build-backend
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux go build -o pocketbase .
RUN mkdir /output && mv /app/pocketbase /output
COPY --from=build-frontend /app/dist/assets/index.js /output/pb_public/comment.js

FROM alpine:latest AS production
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY --from=build-backend /output .
EXPOSE 8090
ENTRYPOINT ["./pocketbase", "serve", "--http=0.0.0.0:8090"]

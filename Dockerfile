################################################
# Build the frontend
################################################

FROM node:14.11.0-alpine as frontend_builder

WORKDIR /app

# install and cache app dependencies
COPY web/package.json ./
COPY web/yarn.lock ./
RUN yarn install --frozen-lockfile

# bundle app source inside Docker image
COPY web/. .
RUN yarn run build

FROM golang:1.16-alpine AS server_builder

WORKDIR /app

COPY go.* ./

# install dependencies
RUN go mod download

# copy built frontend static files
COPY --from=frontend_builder ./app/build ./web/build

ARG VERSION

# add all other folders required for the Go build
COPY . .
COPY web/build.go web/build.go

RUN go build -ldflags "-X commands.version=$VERSION" -o bin/iamzero-all-in-one cmd/all-in-one/main.go

FROM alpine:3.13.5

WORKDIR /app

COPY --from=server_builder /app/bin/iamzero-all-in-one /app/iamzero-all-in-one

# set HTTP ingress port
ENV IAMZERO_COLLECTOR_HOST=0.0.0.0:13991
ENV IAMZERO_CONSOLE_HOST=0.0.0.0:14321

# Web HTTP
EXPOSE 14321

# Collector HTTP
EXPOSE 13991

# Healthcheck
EXPOSE 10866

CMD /app/iamzero-all-in-one

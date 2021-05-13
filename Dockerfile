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

# add all other folders required for the Go build
COPY . .
COPY web/build.go web/build.go

RUN go build -o bin/iamzero-server cmd/main.go

FROM alpine:3.13.5

WORKDIR /app

COPY --from=server_builder /app/bin/iamzero-server /app/iamzero-server

# set HTTP ingress port to standard port 80
ENV IAMZERO_HOST=0.0.0.0:80 

EXPOSE 80

CMD /app/iamzero-server server

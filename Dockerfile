FROM debian:buster-slim as terraform_installer

################################
# Install Terraform
################################
WORKDIR /app

# Install dependencies needed to download and verify Terraform
RUN  apt-get update \
  && apt-get install -y wget gnupg unzip \
  && rm -rf /var/lib/apt/lists/*

# Download terraform for linux
RUN wget https://releases.hashicorp.com/terraform/0.14.11/terraform_0.14.11_linux_amd64.zip
RUN wget https://releases.hashicorp.com/terraform/0.14.11/terraform_0.14.11_SHA256SUMS
RUN wget https://releases.hashicorp.com/terraform/0.14.11/terraform_0.14.11_SHA256SUMS.sig

COPY docker/hashicorp.asc ./

# Verify the signature file is untampered.
RUN gpg --import hashicorp.asc

RUN gpg --verify terraform_0.14.11_SHA256SUMS.sig terraform_0.14.11_SHA256SUMS

# Verify the SHASUM matches the archive.
RUN sha256sum --ignore-missing -c terraform_0.14.11_SHA256SUMS 

# Unzip
RUN unzip terraform_0.14.11_linux_amd64.zip

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

# add the terraform binary
COPY --from=terraform_installer /app/terraform /usr/local/bin/terraform

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

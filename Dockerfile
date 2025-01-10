# syntax=docker/dockerfile:1

FROM golang:1.23

# Set destination for COPY
WORKDIR /app
RUN apt-get update && apt-get install -y make
# Download Go modules
COPY . .
RUN go get .
# Force color output
ENV TERM=xterm-256color
ENV FORCE_COLOR=1
# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy

# Build
# RUN CGO_ENABLED=0 GOOS=linux go build -o /docker-gs-ping

# Run
CMD ["make", "tests-verbose"]
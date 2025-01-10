FROM golang:1.23

WORKDIR /app
RUN apt-get update && apt-get install -y make
# Download Go modules
COPY . .
RUN go get .

CMD ["make", "tests-verbose"]
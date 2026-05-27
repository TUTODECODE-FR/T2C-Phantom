FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o t2c cmd/t2c/main.go

# Start script
RUN echo '#!/bin/sh' > start.sh
RUN echo 'mkdir -p data' >> start.sh
RUN echo 'if [ ! -f data/key ]; then ./t2c genkey | grep "Private Key" | awk "{print \$3}" > data/key; fi' >> start.sh
RUN echo 'KEY=$(cat data/key)' >> start.sh
RUN echo 'if [ -z "$BOOTSTRAP" ]; then' >> start.sh
RUN echo '  ./t2c start --key $KEY --port $PORT' >> start.sh
RUN echo 'else' >> start.sh
RUN echo '  ./t2c start --key $KEY --port $PORT --bootstrap $BOOTSTRAP' >> start.sh
RUN echo 'fi' >> start.sh
RUN chmod +x start.sh

CMD ["./start.sh"]

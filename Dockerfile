FROM golang:latest
WORKDIR /app
COPY . .
COPY assets/style.css app/assets/
RUN go get -d -v
RUN go build -o main .
EXPOSE 8081
CMD ["./main"]
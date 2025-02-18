FROM golang:latest
WORKDIR /app

COPY . .

CMD ["bash"]

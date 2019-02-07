FROM golang:1.11 as build
# RUN apt-get update && apt-get install -y upx-ucl
RUN mkdir -p /config
ADD go.mod go.sum /src/transmissio/
WORKDIR /src/transmissio/
RUN go mod download

COPY . /src/transmissio/
RUN CGO_ENABLED=0 GOOS=linux go build -o transmissio
# RUN upx transmissio

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /config /config
COPY --from=build /src/transmissio/transmissio /bin/transmissio
WORKDIR /config
ENTRYPOINT ["/bin/transmissio"]
EXPOSE 9091

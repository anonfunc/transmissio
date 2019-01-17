FROM golang:1.11 as build

RUN mkdir -p /config
ADD go.mod go.sum /src/transmissio/
WORKDIR /src/transmissio/
RUN go mod download

COPY . /src/transmissio/
RUN CGO_ENABLED=0 GOOS=linux go build -o transmissio

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /src/transmissio/transmissio /bin/transmissio
COPY --from=build /config /config
WORKDIR /config
ENTRYPOINT ["/bin/transmissio"]
EXPOSE 9091

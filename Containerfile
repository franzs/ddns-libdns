FROM golang:1.24 as build

ARG DDNS_PROVIDER_TAG

WORKDIR /app
COPY . .

RUN go mod download
RUN go vet -tags "${DDNS_PROVIDER_TAG}" -v

RUN CGO_ENABLED=0 go build -tags "${DDNS_PROVIDER_TAG}" -o ddns-libdns

FROM gcr.io/distroless/static-debian12

COPY --from=build /app/ddns-libdns /

EXPOSE 8080
CMD ["/ddns-libdns"]

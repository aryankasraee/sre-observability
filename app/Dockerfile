FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/shop-api .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/shop-api /shop-api
EXPOSE 8080
ENTRYPOINT ["/shop-api"]

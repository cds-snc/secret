FROM golang:1.24.3-alpine@sha256:ef18ee7117463ac1055f5a370ed18b8750f01589f13ea0b48642f5792b234044 as build

ARG component=${component}

ENV USER=app
ENV UID=10001

WORKDIR /app

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

COPY . .
RUN go build -o /server /app/cmd/${component}/main.go

FROM alpine:latest@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c as certs
RUN apk --update add ca-certificates

FROM scratch 

ARG GIT_SHA

ENV USER=app
ENV GIT_SHA=${GIT_SHA}

ENV PATH=/bin
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build --chown=${USER}:${USER} /server /server
COPY --from=build --chown=${USER}:${USER} /app/keys /keys
COPY --from=build --chown=${USER}:${USER} /app/locales /locales
COPY --from=build --chown=${USER}:${USER} /app/views /views

USER ${USER}:${USER}

ENTRYPOINT ["/server"]

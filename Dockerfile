FROM golang:1.25.3-alpine@sha256:aee43c3ccbf24fdffb7295693b6e33b21e01baec1b2a55acc351fde345e9ec34 as build

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

FROM alpine:latest@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412 as certs
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

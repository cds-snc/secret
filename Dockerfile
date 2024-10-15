FROM golang:1.23.2-alpine@sha256:9dd2625a1ff2859b8d8b01d8f7822c0f528942fe56cfe7a1e7c38d3b8d72d679 as build

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

FROM alpine:latest@sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d as certs
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

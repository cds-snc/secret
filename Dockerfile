FROM golang:1.23.5-alpine@sha256:47d337594bd9e667d35514b241569f95fb6d95727c24b19468813d596d5ae596 as build

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

FROM alpine:latest@sha256:56fa17d2a7e7f168a043a2712e63aed1f8543aeafdcee47c58dcffe38ed51099 as certs
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

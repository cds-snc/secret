FROM golang:1.22.5-alpine@sha256:0d3653dd6f35159ec6e3d10263a42372f6f194c3dea0b35235d72aabde86486e as build

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

FROM alpine:latest@sha256:0a4eaa0eecf5f8c050e5bba433f58c052be7587ee8af3e8b3910ef9ab5fbe9f5 as certs
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

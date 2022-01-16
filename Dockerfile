FROM alpine:3.15 AS builder
LABEL author="Alexander Schenkel <alex@alexi.ch>"

RUN apk update && apk add \
	make \
	musl-dev \
	go

# Configure Go
ENV GOROOT /usr/lib/go
ENV GOPATH /go
ENV PATH /go/bin:$PATH
ENV PCMS_PATH /pcms

RUN mkdir -p ${GOPATH}/src ${GOPATH}/bin
RUN mkdir -p ${PCMS_PATH}

COPY src/ ${PCMS_PATH}/src/
COPY main.go ${PCMS_PATH}/
COPY go.mod ${PCMS_PATH}/
COPY go.sum ${PCMS_PATH}/
COPY Makefile ${PCMS_PATH}/

WORKDIR ${PCMS_PATH}

RUN make build

# -------------
FROM alpine:3.15 AS pcms
LABEL author="Alexander Schenkel <alex@alexi.ch>"
EXPOSE 3000

ENV PCMS_PATH /pcms
ENV PATH "${PATH}:/pcms/bin"
RUN mkdir -p ${PCMS_PATH}/bin ${PCMS_PATH}/doc/log ${PCMS_PATH}/site-template
COPY --from=builder ${PCMS_PATH}/bin/pcms ${PCMS_PATH}/bin/pcms
COPY site-template/ ${PCMS_PATH}/site-template/
COPY doc/ ${PCMS_PATH}/doc/
RUN rm -rf ${PCMS_PATH}/doc/log/*
RUN chmod a+w ${PCMS_PATH}/doc/log
RUN addgroup pcms
RUN adduser -D -s /bin/sh -G pcms pcms

USER pcms

WORKDIR ${PCMS_PATH}/doc
CMD pcms
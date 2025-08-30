FROM alpine:3.18 AS builder
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

COPY ./ ${PCMS_PATH}/

WORKDIR ${PCMS_PATH}

RUN make build

# -------------
FROM alpine:3.18 AS pcms
LABEL author="Alexander Schenkel <alex@alexi.ch>"
EXPOSE 3000

ENV PCMS_PATH /pcms
ENV PATH "${PATH}:/pcms/bin"
COPY --from=builder ${PCMS_PATH}/bin/pcms ${PCMS_PATH}/bin/pcms
RUN addgroup pcms
RUN adduser --disabled-password --shell /bin/sh --ingroup pcms pcms

USER pcms

WORKDIR ${PCMS_PATH}
CMD pcms serve-doc
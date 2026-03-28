FROM alpine:3.23 AS builder
LABEL author="Alexander Schenkel <alex@alexi.ch>"

RUN apk update && apk add \
	make \
	musl-dev \
	go

# Configure Go
ENV GOROOT=/usr/lib/go
ENV GOPATH=/go
ENV PATH=/go/bin:$PATH
ENV PCMS_PATH=/pcms

RUN mkdir -p ${GOPATH}/src ${GOPATH}/bin
RUN mkdir -p ${PCMS_PATH}

COPY ./ ${PCMS_PATH}/

WORKDIR ${PCMS_PATH}

RUN make build

# -------------
FROM alpine:3.23 AS pcms
LABEL author="Alexander Schenkel <alex@alexi.ch>"
EXPOSE 3000

ENV PCMS_PATH=/pcms
ENV PATH="${PATH}:/pcms/bin"
COPY --from=builder ${PCMS_PATH}/bin/pcms ${PCMS_PATH}/bin/pcms
RUN addgroup pcms
RUN adduser --disabled-password --shell /bin/sh --ingroup pcms pcms

WORKDIR ${PCMS_PATH}
RUN chown -R pcms:pcms .

USER pcms
CMD ["pcms", "serve-doc"]
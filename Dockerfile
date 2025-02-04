FROM golang:1.22.11 AS base

WORKDIR /

COPY ./ /

RUN go build -tags netgo,osusergo . 

FROM scratch

COPY --from=base /cc-intel-platform-registration /bin/cc-intel-platform-registration

ENTRYPOINT ["/bin/cc-intel-platform-registration"]

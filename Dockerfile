FROM golang:1.6.1

RUN mkdir -p /usr/bin

ADD ./netPing /usr/bin

ENTRYPOINT ["/usr/bin/netPing"]
CMD ["start"]

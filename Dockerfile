FROM acs-reg.alipay.com/acs/acs-perf:latest

RUN mkdir -p /usr/bin

ADD ./netPing /usr/bin

ENTRYPOINT ["/usr/bin/netPing"]
CMD ["start"]

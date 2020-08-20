# Build stage
FROM golang:latest as build-stage
WORKDIR /app
COPY . .
RUN make deps
RUN make lininfosec

# Production stage
FROM busybox:glibc
COPY --from=build-stage /app/build/bin/lininfosec ./lininfosec
RUN mkdir /data
ENV LININFOSEC_DATA_DIR=/data
EXPOSE 9999
CMD ["./lininfosec"]

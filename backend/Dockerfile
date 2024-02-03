# SPDX-FileCopyrightText: Free Software Foundation Europe <https://fsfe.org>
#
# SPDX-License-Identifier: AGPL-3.0-or-later

# =============================================================================
# Development-only container - will NOT be used for production
#
# Can be used for interactive debuging:
# $ docker build --target backend-dev -t backend-dev .
# $ docker run -it --rm -p 8080:8080 -v "$PWD":/app backend-dev sh
# =============================================================================

FROM golang:1.21-alpine3.18 AS backend-dev

RUN apk add --no-cache build-base imagemagick-dev imagemagick strace

WORKDIR /app

COPY inc/fonts /usr/share/fonts/sharepic-fonts
RUN fc-cache

COPY inc/policy.xml /etc/ImageMagick-7/policy.xml


# =============================================================================
# Container to build the backend
# =============================================================================

FROM golang:1.21-alpine3.18 AS backend-builder

RUN apk add --no-cache build-base imagemagick-dev imagemagick

WORKDIR /app
COPY . .

ENV CGO_ENABLED=1
ENV CGO_CFLAGS_ALLOW=-Xpreprocessor
RUN go mod download && \
    go build -o /backend


# =============================================================================
# Backend contaner to be used
# =============================================================================

FROM alpine:3.18

RUN apk --no-cache add curl imagemagick-dev imagemagick

COPY inc/fonts /usr/share/fonts/sharepic-fonts
RUN fc-cache

COPY inc/policy.xml /etc/ImageMagick-7/policy.xml

COPY --from=backend-builder /backend /bin/backend

RUN addgroup -S app && adduser -S -G app app
USER app

CMD /bin/backend

HEALTHCHECK CMD curl -f http://localhost:8080/health || exit 1

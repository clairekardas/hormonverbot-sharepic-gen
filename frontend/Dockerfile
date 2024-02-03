# SPDX-FileCopyrightText: Free Software Foundation Europe <https://fsfe.org>
#
# SPDX-License-Identifier: AGPL-3.0-or-later

FROM bitnami/apache:2.4 AS development

# Install curl for the HEALTHCHECK below.
USER 0
RUN install_packages curl
USER 1001

# Remove the default VHost and install our custom one.
RUN rm /opt/bitnami/apache/conf/vhosts/00_status-vhost.conf
COPY frontend.vhost.conf /vhosts/

# Remove IP and userid from logs.
RUN sed -i -E 's/(\s*)LogFormat "%h %l %u (.+)$/\1LogFormat "\2/g' /opt/bitnami/apache/conf/httpd.conf

HEALTHCHECK CMD curl -f http://localhost:8080/health/frontend || exit 1

# =============================================================================
# Production installation
# =============================================================================

FROM development AS production

# Copy the frontend files that are just mounted in development mode

COPY www/ /app/

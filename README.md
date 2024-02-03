<!--
SPDX-FileCopyrightText: Free Software Foundation Europe <https://fsfe.org>

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Sharepic Generator

[![in docs.fsfe.org](https://img.shields.io/badge/in%20docs.fsfe.org-OK-green)](https://docs.fsfe.org/repodocs/sharepic/00_README)
[![Build Status](https://drone.fsfe.org/api/badges/fsfe-system-hackers/sharepic/status.svg)](https://drone.fsfe.org/fsfe-system-hackers/sharepic)
[![REUSE status](https://api.reuse.software/badge/git.fsfe.org/fsfe-system-hackers/sharepic)](https://api.reuse.software/info/git.fsfe.org/fsfe-system-hackers/sharepic)

A sharepic - compound of _to share_ and _picture_ - is a themed picture to communicate a quote of a person.
This repository contains a web application to generate such sharepics for

- [#ilovefs](https://fsfe.org/activities/ilovefs/index.en.html),
- [Public Money? Public Code!](https://publiccode.eu/), and
- [SFScon - Free Software Conference](https://www.sfscon.it/).

## Getting Started
To run this Sharepic Generator on your own machine, Docker and Docker Compose are required.
The software can be started for local development via:
```
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```
After building the containers is completed and they are running, the web application is accessible at <http://localhost:8830/>.

## Architecture
The application consists of two components: a frontend and a backend.
On the frontend, an HTTP server provides an input mask that can be used to create sharepics.
Then the backend receives this data and creates the sharepic, which is delivered through the frontend.

To simplify deployment, Docker Compose is a key component.
This completely takes care of creating the necessary containers, isolating them and allowing them to communicate with each other.

### Frontend
The frontend is an Apache web server, which delivers the web page with the input mask as static files.
To communicate with the backend, the Apache httpd proxies the corresponding HTTP POST request to the backend.

Since the web page does not require any compilation, it is mounted read-only as a volume into the frontend container in the local development mode (using the `-f docker-compose.dev.yml` argument).
Thus, changes can be made live without restarting.
In production mode, it will be copied into the container.

### Backend
The backend consists of an application written for this purpose in the Go programming language, which also starts an HTTP server.
On the `/sharepic` endpoint it expects the POST request of the frontend and creates the sharepic accordingly.

Technically, the SVG image file is used as a template and modified by the sanitized input data.
Then ImageMagick is used to convert the customized SVG to a PNG file.

All configuration is achieved by the YAML files within `backend/inc`.
While the `backend/inc/backend.yml` file defines global settings, a specific `.yml` file for each template in `backend/inc/templates` configures template related settings, i.e. the font.

## Backend Development Container
An interactive development container for the backend is specified within the `backend/Dockerfile`.

It can be created and entered as follows from the repository's root:
```
docker build --target backend-dev -t backend-dev backend
docker run -it --rm -p 8080:8080 -v "$PWD/backend":/app backend-dev sh
```

As the `backend` directory being bind mounted, one can make changes on the host system.
Within the container shell, a new backend can be built and tested:
```
go build
./backend
```

If nothing has failed, the backend's HTTP server can be reached on the host via <http://localhost:8080/>.
As this HTTP server does not contain the frontend, one can, e.g., use `curl` to generate a sharepic:
```
curl -F 'img=@/tmp/gnu.jpg' -F 'template=ilovefs' -F 'message=#iLoveFs' 'http://localhost:8080/sharepic'
```

## New Template
### Backend Part
A template consists of two identically named files - one with the `.svg` and one with the `.yml` extension - within the `backend/templates` directory.
Creating those two files is all it takes for adding a new template.

During the backend software's compile time, those files are sourced and included in the binary.
When starting up, those files are parsed and prepared within the program's memory.

#### Customized SVG file
To use a SVG graphic as the base, some manual modifications needs to be performed to the file.
It's recommended to use a text editor therefore.

- Create a pseudo-`image` tag like the following one where the user's image should appear:
  ```
  <image . . . xlink:href="data:image/jpeg;base64,{{.ImageData}}" />
  ```
- Replace the one-line text for the user's name with: `{{.AuthorName}}`.
- Replace the one-line text for the user's position or description with: `{{.AuthorDesc}}`.

#### YAML Configuration
_Note:_ The file extension must be `.yml`, not `.yaml`.

The YAML file describes both the created sharepic as well as the parameters for the multi-lined user message.
A described example configuration follows.

```yaml
# Resulting sharepic's dimensions.
sharepic:
  width: 640
  height: 480

# Size the user submitted picture should be reduced to.
# This value should roughly match the image tag within the SVG.
# The somewhat custom grayscale tag allows converting the picture to grayscale.
picture_box:
  width: 80
  height: 87
  grayscale: no

# Font to be used for the message - must be installed on the system.
# The font color might be specified as in HTML, e.g., as black or #ffffff.
# To transform all user input to uppercase, uppercase can be set.
# The font sizes will all be tried, the biggest possible one will be used.
font:
  name: Liberation-Sans
  color: black
  uppercase: no
  sizes:
    - 22
    - 26
    - 30
    - 34

# Maximum amount of characters for the message, the author, and its description.
max_length:
  message: 200
  author: 50
  description: 50

# The message will appear within an (invisible) box with its own dimensions,
# which must be smaller than those of the image.
# The margin defines the offset from the top-left point of the image.
# Setting disable results in a sharepic without an overlay text.
message_box:
  disable: no

  width: 289
  height: 220

  margin_width: 27
  margin_height: 72
```

### Frontend Part
As the frontend being just one single `index.html` file, another template can be added as a new
```
<input . . . name="template" value="__TEMPLATE_NAME__" />
```
entry next to the other ones.
An exemplary image should be placed as `frontend/www/assets/examples/__TEMPLATE_NAME__.jpg`.

Furthermore, to make the redirections work, the `RewriteRule` within `frontend/frontend.vhost.conf` must be extended.

## Security Considerations
The most critical component would seem to be ImageMagick, which has a [reputation for security issues](https://cve.mitre.org/cgi-bin/cvekey.cgi?keyword=imagemagick).
Thus, ImageMagick is limited to the necessary modules by a custom [Security Policy](https://imagemagick.org/script/security-policy.php) in the `backend/inc/policy.xml` file.
Further, the backend software also checks that only allowed file types can be uploaded.

The software in the backend container runs with limited user rights and a `seccomp-bpf` filter applied.
Also, the container is not in the default network, so no outgoing connections to the Internet should be possible.
From the outside, the backend container should not be accessible, but only the `/sharepic` HTTP endpoint through the Apache reverse proxy in the frontend.

However, the frontend container is also configured to run the Apache web server with limited user rights.

There is no bidirectional writable volume between container and host.
The frontend has read-only rights to the data on the web page.
The backend stores temporary files in a volatile manner, which it deletes immediately after each operation.

## Privacy Considerations
Despite the fact that the software is based on a typical client-server model, efforts were made to minimize the amount of data generated and not to store it anywhere permanently.

The Apache HTTP server in the frontend was configured to not log personal data like IP addresses.

Both the backend software's architecture as well as its deployment is built with the privacy aspect in mind.
No personal data should be logged and nothing is stored on persistent storage.

More details regarding this are present in [issue #1](https://git.fsfe.org/fsfe-system-hackers/sharepic/issues/1).

## License
This program is [REUSE](https://reuse.software) compliant.
Each file contains or is accompanied by licensing and copyright information.
The majority of the code is licensed under AGPL-3.0-or-later.

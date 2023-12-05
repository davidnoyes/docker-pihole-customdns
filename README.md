# Docker Pi-hole Custom DNS

Facilitate Pi-hole DNS entry automation for Docker services efficiently with docker-pihole-customdns. Simplify the management process, ensuring seamless integration and reduced manual effort.

Perfect for use when running your services behind a proxy such as Traefik or nginx. Instead of manually adding and removing DNS entries for your self-hosted Docker applications in Pi-hole, specify the domain to use in a label on your container and let Docker Pi-hole Custom DNS create the DNS entry for you.

## Supports

* DNS A record management
* DNS CNAME record management
* Up to two Pi-hole DNS servers

## Docker Image

```shell
docker pull download.noyes.uk/davidnoyes/docker-pihole-customdns:latest
```

*(Images hosted by ghcr.io with a custom domain)*

**OS / Arch**

* linux/amd64
* linux/arm/v7
* linux/arm64

## Docker Usage

```shell
docker run --name docker-pihole-customdns -d --restart=unless-stopped -v /var/run/docker.sock:/var/run/docker.sock:ro -e DPC_PIHOLE_URL=http://pi.hole -e DPC_DOCKER_HOST_IP=198.51.100.0 -e DPC_PIHOLE_API_TOKEN=abcdefghijklmnopqrstuvwxyz
```

Replace the values for `DPC_PIHOLE_URL`, `DPC_DOCKER_HOST_IP` & `DPC_PIHOLE_API_TOKEN` as appropriate

## Docker Label

The conatiner label to be applied to service containers is: 

`docker-pihole-customdns.domain=`

It can be applied in the following way:
```shell
docker run -d --name nginx -l docker-pihole-customdns.domain=my-service.int.my-domain.net nginx
```

## Docker Compose

```yaml
version: "3.8"
services:
  docker-pihole-customdns:
    container_name: docker-pihole-customdns
    image: ghcr.io/davidnoyes/docker-pihole-customdns:latest
    restart: unless-stopped
    security_opt:
      - no-new-privileges:true
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      - DPC_PIHOLE_API_TOKEN=abcdefghijklmnopqrstuvwxyz
      - DPC_DOCKER_HOST_IP=198.51.100.0
      - DPC_PIHOLE_URL=http://pi.hole
```

### Environment Variables

| Variable | Description |
|-|-|
| `DPC_PIHOLE_API_TOKEN` | Pi-hole API Token. |
| `DPC_PIHOLE_API_TOKEN_2` | Second Pi-hole API Token (Optional)
| `DPC_DOCKER_HOST_IP` | Docker host IP address. The IP address used by the http proxy for all docker services on the host. |
| `DPC_PIHOLE_URL` | Pi-hole URL (e.g. http://pi-hole) |
| `DPC_PIHOLE_URL_2` | Second Pi-hole URL (optional) |

## Binary Usage

```shell
Usage of ./docker-pihole-customdns:
  -apitoken string
        Pi-hole API token
  -apitoken2 string
        Second Pi-hole API token (Optional)
  -hostip string
        Docker host IP address
  -piholeurl string
        Pi-hole URL (e.g. http://pi.hole)
  -piholeurl2 string
        Second Pi-hole URL (optional e.g. http://pi.hole)
```


## Refrences

### Pi-hole DNS API

Pi-hole API token can be obtained here: http://pi.hole/admin/settings.php?tab=api
(Where `pi.hole` will resolve to your Pi-hole instance or replace with your own domain/IP address.)

[CustomDNS](docs/CustomDNS.md)

[CustomCNAME](docs/CustomCNAME.md)

<img referrerpolicy="no-referrer-when-downgrade" src="https://static.scarf.sh/a.png?x-pxid=d392e053-6689-4d55-98a0-70f0ed688db1" />

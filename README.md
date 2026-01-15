# ddns-libdns

Dynamic DNS service for usage with libdns which uses the [dyndns v3](https://help.dyn.com/perform-update.html) prototol.

## Configuration

ddns-libdns is configured with environment variables:

### `DDNS_AUTH_CONFIG`

This example shows the structure of the auth configuration:

```json
[
  {
    "username": "user1",
    "passwordHash": "$argon2id$v=19$m=4096,t=3,p=1$YnBnQnRxN1hmUS9TTWd3Sm5Fc3kyZz09$r1pfigbhl/Wisa1yYQ87Etv0WSPpRL84Q/K23aMM5f0",
    "hostnames": [
      "host1.domain1.com",
      "host1.domain1.com"
    ]
  },
  {
    "username": "user2",
    "passwordHash": "$argon2id$v=19$m=4096,t=3,p=1$VnZOVDVMSlJOSGJ5QS8zckVBMy96Zz09$avW/NBu+ZbWJhQI4Eq9lC6Agk0exPkbo/19wMDvhJZY",
    "hostnames": [
      "host2.domain2.com",
      "host2.domain2.com"
    ]
  }
]
```

Basically it is a list of users with a password hash and hostnames the user is allowed to change. The password hash can take any content [`crypt`](https://github.com/go-crypt/crypt) can digest. To generate a password hash with the `argon2id` algorithm use something like this:

```shell
echo -n "secret" | argon2 "$(openssl rand -base64 16)" -id -e
```

For other algorithms [`mkpasswd`](https://manpages.ubuntu.com/manpages/trusty/man1/mkpasswd.1.html) can be used:

```shell
mkpasswd --method bcrypt secret
```

Assuming your auth configuration is saved in a file named `auth.json` you can compact it with `jq`:

```shell
export DDNS_AUTH_CONFIG=$(jq -c . < auth.json)
```

### Authentication for DNS provider

Set one of following environment variables according to the DNS provider you'd like to use.

* `DDNS_BUNNY_ACCESSKEY`
* `DDNS_DESEC_TOKEN`
* `DDNS_HETZNER_TOKEN`
* `DDNS_IONOS_APITOKEN`

## Usage

Send a GET request with HTTP basic access authentication and the GET parameters `hostname` and `myip`. `myip` takes one or more IPv4 and IPv6 addresses. Multiple IP addresses have to be separated by comma:

```shell
curl --user user1:secret 'http://localhost:8080/v3/update?hostname=host1.domain.com&myip=192.168.123.1'
```

## Usage with Fritz!BOX

Enter something like this as `Update URL`:

```
https://dyndns.example.com/v3/update?hostname=<domain>&myip=<ip4addr>,<ip6addr>
```

`Domain name ` is your hostname you'd like to update. `User name` and `Password` correspond to your `DDNS_AUTH_CONFIG`.

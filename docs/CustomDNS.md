# Pi-hole DNS API
# Get Custom DNS

**URL** : `/admin/api.php?customdns`

**Method** : `GET`

**Auth** : `&auth=API_KEY`

**Action** : `&action=get`

**Params** : None

## Success Response

**Code** : `200 OK`

**Content example**
```json
{
  "data": [
    [
      "test.local",
      "127.0.0.1"
    ]
  ]
}
```

# Add Custom DNS

**URL** : `/admin/api.php?customdns`

**Method** : `GET`

**Auth** : `&auth=API_KEY`

**Action** : `&action=add`

**Params** : 

ip: `&ip=127.0.0.1`

domain: `&domain=DOMAIN_NAME`

## Success Response

**Code** : `200 OK`

**Content**
```json
{
    "success": true,
    "message": ""
}
```

## Error Response

**Condition** : If 'ip' and 'domain' combination already exists.

**Code** : `200 OK`

**Content** :

```json
{
  "success": false,
  "message": "The domain test.local already has a custom DNS entry for an IPv4"
}
```

# Delete Custom DNS

**URL** : `/admin/api.php?customdns`

**Method** : `GET`

**Auth** : `&auth=API_KEY`

**Action** : `&action=delete`

**Params** : 

ip: `&ip=127.0.0.1`

domain: `&domain=DOMAIN_NAME`

## Success Response

**Code** : `200 OK`

**Content**
```json
{
    "success": true,
    "message": ""
}
```

## Error Response

**Condition** : If 'ip' and 'domain' combination do not exist.

**Code** : `200 OK`

**Content** :

```json
{
  "success": false,
  "message": "This domain/ip association does not exist"
}
```

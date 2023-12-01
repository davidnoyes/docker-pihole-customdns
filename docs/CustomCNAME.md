# Pi-hole DNS API
# Get Custom DNS

**URL** : `/admin/api.php?customcname`

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
      "testcname.local",
      "test.local"
    ]
  ]
}
```

# Add Custom DNS

**URL** : `/admin/api.php?customcname`

**Method** : `GET`

**Auth** : `&auth=API_KEY`

**Action** : `&action=add`

**Params** : 

domain: `&domain=testcname.local`

target: `&targer=test.local`

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

**Condition** : If 'domain' and 'target' combination already exists.

**Code** : `200 OK`

**Content** :

```json
{
  "success": false,
  "message": "There is already a CNAME record for 'testcname.local'"
}
```

# Delete Custom DNS

**URL** : `/admin/api.php?customcname`

**Method** : `GET`

**Auth** : `&auth=API_KEY`

**Action** : `&action=delete`

**Params** : 

domain: `&domain=testcname.local`

target: `&targer=test.local`

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

**Condition** : If 'domain' and 'target' combination do not exist.

**Code** : `200 OK`

**Content** :

```json
{
  "success": false,
  "message": "This domain/ip association does not exist"
}
```

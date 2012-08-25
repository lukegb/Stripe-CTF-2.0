Level 3 notes (level03)
=======================

Ah yes, the Secret Vault. This Python software is something you don't have the password for, so you need to figure out how you're going to break in from the get-go.

What's the vulnerability?
-------------------------

This code:
```python
    query = """SELECT id, password_hash, salt FROM users
               WHERE username = '{0}' LIMIT 1""".format(username)
```

Instead of using prepared statements, Python's string interpolation is used to insert the username. Oops. This means that there's an SQL injection vulnerability!

How do you exploit it?
----------------------

You submit a username like so:

```
' UNION SELECT 1, '2c93a9e75e05682d26687d2799a47f3f38a138f3a7b31d01a8f9470ace0fc0e9', 'saltysalt' --
```

The 1 indicates that you're logging in with user ID 1, which I presumed would be the user with the level password.
The hex string is the SHA256 hash of "passwordsaltysalt", allowing you to log in with the password 'password'.

Easy.
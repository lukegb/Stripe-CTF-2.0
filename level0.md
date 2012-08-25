Level 0 (level0) notes
======================

This level involves the Secret Safe, a node.js and SQLite3 application.

What's the vulnerability?
-------------------------

This line:
```javascript
var query = 'SELECT * FROM secrets WHERE key LIKE ? || ".%"';
```

How do you exploit it?
----------------------

Enter %, the SQL LIKE wildcard character in the "view secrets for" textbox and the password will magically appear.
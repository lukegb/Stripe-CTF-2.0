Level 1 (level01) notes
=======================

This level involves a small PHP script with a nice, traditional vulnerability.

What **is** the vulnerability?
------------------------------

This line:
```php
extract($_GET);
```

That line is very reminiscent of the days in which register_globals used to be turned on in PHP. It takes everything you've sent as a parameter and makes it a variable.
Sending ?p=2 would set the variable $p to 2.

How do you exploit it?
----------------------

You can override the filename parameter with something else. I chose /dev/null because I knew it would immediately return an empty file, and it was something I thought the server would have access to.

Therefore, your query string should be
`?filename=/dev/null&attempt=`
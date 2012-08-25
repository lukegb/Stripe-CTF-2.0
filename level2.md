Level 2 notes (level02)
=======================

This level involves a pseudo-social-network which only has a display image upload function.

This level also forms the base for solving two other levels. :)

What's the vulnerability?
-------------------------

The upload mechanism doesn't check to see what you've uploaded, and PHP is enabled inside the uploads directory.

How do you exploit it?
----------------------

The simplest way of exploiting it is:
```php
echo file_get_contents("../password.txt");
```

and uploading that as get.php, then browsing to uploads/get.php. The password will be displayed for you to move on to the next level.
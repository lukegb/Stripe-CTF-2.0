Level 6 notes (level06)
=======================

This level involves "Streamer", a cheap and quick not-quite-Twitter.

What's the vulnerability?
-------------------------

You can inject JavaScript into the Streams you publish.

What's the complication?
------------------------

' and " are blacklisted, and make up part of the user you're trying to hack's password.
There are two ways of getting around this: replace ' and " with other characters, or base64 encode everything (I opted for this).

This also means that either you use data URLs (base64 again!):

```html
</script><script src=data:application/javascript;base64,YWxlcnQoIllvdSd2ZSBiZWVuIFBXTkVEISIpOw==></script><script>//
```

Or you construct JavaScript with no quotes, like so:

```html
</script>
var stringWithoutQuotes = String.fromCharCode(72, 101, 108, 108, 111, 33);
// do stuff with stringWithoutQuotes
<script>//
```

Alternatively, you can use this method and eval your entire script:

```html
</script>
eval(String.fromCharCode(97, 108, 101, 114, 116, 40, 34, 80, 87, 78, 69, 68, 33, 34, 41, 59));
<script>//
```

Websites like http://jdstiles.com/java/cct.html are helpful in encoding your JavaScript.
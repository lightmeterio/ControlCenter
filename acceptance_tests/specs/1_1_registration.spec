# Registration

## Registration fails
Tags: failure

* Go to registration page
* Focus on field with placeholder "Name"
* Type "User Complete Name"
* Focus on field with placeholder "Email"
* Type "acceptance_tests@lightmeter.io"
* Focus on field with placeholder "Password"
* Type "54353%#%#54354353gffgdgdfg"
* Expect registration to fail
* Click on "Register"

## Registration succeeds
Tags: success

* Go to registration page
* Focus on field with placeholder "Name"
* Type "User Complete Name"
* Focus on field with placeholder "Email"
* Type "acceptance_tests@lightmeter.io"
* Focus on field with placeholder "Password"
* Type "54353%#%#54354353gffgdgdfg"
* Click on "Register"
* Expect to be in the main page

# Usage with docker

First, build docker image:

```
cd ci
docker build -n lm_builder .
```

Then build ligtmeter with:

```
docker run -v /path/to/lightmeter/directory:/src -v/path/to/output/directory:/dst lightmeter-builder
```

You'll then get a file called `lightmeter` in the directory `/path/to/output/directory/`.

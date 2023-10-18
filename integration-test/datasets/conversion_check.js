db.createCollection(
    "conversion_check",
);


db.conversion_check.insertMany( [
    {
        "int": 1,
        "float": 2.5,
        "string": "Hello, world!",
        "bool": true,
        "object": { "foo": 1, "bar": 2.5, "baz": "Hello, world!" },
        "array": [1,2.5,"Hello, World!"],
        "datetime": Date(),
        "timestamp": Timestamp({"t": 0, "i": 0}),
        "decimal": Decimal128("10000.1"),
    },
]);


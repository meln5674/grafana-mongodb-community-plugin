use non_default_auth_source;

db.test.insert({"test": 1});

db.createUser({
    user: "test-user",
    pwd: "test-password",
    roles: [ 
        { role: "readWrite", db: "test"},
        { role: "readWrite", db: "twitter"},
    ],
});

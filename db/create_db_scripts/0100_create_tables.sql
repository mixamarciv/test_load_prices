CREATE TABLE itemtype (
    id            INTEGER,
    name          VARCHAR(2000)
);

CREATE UNIQUE INDEX itemtype_IDX2 ON itemtype (id);
CREATE        INDEX itemtype_IDX3 ON itemtype (name);


CREATE TABLE sell_order (
    id            INTEGER,
    price         BIGINT,
    cnt           INTEGER,
    station       VARCHAR(1000),
    expires       VARCHAR(300)
);

CREATE INDEX sell_order_IDX2 ON sell_order (id,price);

CREATE TABLE buy_order (
    id            INTEGER,
    price         BIGINT,
    cnt           INTEGER,
    station       VARCHAR(1000),
    expires       VARCHAR(300)
);
CREATE INDEX buy_order_IDX2 ON buy_order (id,price);


COMMIT WORK;

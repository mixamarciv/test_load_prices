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


SET TERM ^ ;


/* Trigger: TASK_BI0 */
CREATE TRIGGER itemtype_BI0 FOR itemtype
ACTIVE BEFORE INSERT POSITION 0
AS
begin
  new.date_create = current_timestamp;

end
^


/* Trigger: TASK_BU0 */
CREATE TRIGGER itemtype_BU0 FOR itemtype
ACTIVE BEFORE UPDATE POSITION 0
AS
begin
  new.date_create = old.date_create;
  new.date_update = current_timestamp;
end
^

SET TERM ; ^

COMMIT WORK;

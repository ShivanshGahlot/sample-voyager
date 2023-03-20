-- RANGE PARTITIONS
create table ORDER_ITEMS_RANGE_PARTITIONED (
  order_id        integer
                  generated by default on null as identity,
  order_datetime  timestamp not null,
  customer_id     integer not null,
  order_status    varchar2(10 char) not null,
  store_id        integer not null)
PARTITION BY RANGE (order_id,customer_id)
(PARTITION p1 VALUES LESS THAN (50,8),
PARTITION p2 VALUES LESS THAN (70,15),
PARTITION p3 VALUES LESS THAN (80,15))
ENABLE ROW MOVEMENT;
ALTER TABLE ORDER_ITEMS_RANGE_PARTITIONED ADD (CONSTRAINT ord_cus_pk PRIMARY KEY (order_id,customer_id));

-- LIST PARTITIONS
CREATE TABLE ACCOUNTS_LIST_PARTITIONED
( id             NUMBER
, account_number NUMBER
, customer_id    NUMBER
, branch_id      NUMBER
, region         VARCHAR(2)
, status         VARCHAR2(1)
)
PARTITION BY LIST (region)
( PARTITION p_northwest VALUES ('OR', 'WA')
, PARTITION p_southwest VALUES ('AZ', 'UT', 'NM')
, PARTITION p_northeast VALUES ('NY', 'VM', 'NJ')
, PARTITION p_southeast VALUES ('FL', 'GA')
, PARTITION p_northcentral VALUES ('SD', 'WI')
, PARTITION p_southcentral VALUES ('OK', 'TX')
);
ALTER TABLE ACCOUNTS_LIST_PARTITIONED ADD (CONSTRAINT reg_pk PRIMARY KEY (id, region));

--INTERVAL PARTITIONS
CREATE TABLE ORDERS_INTERVAL_PARTITION
  (
    order_id NUMBER 
             GENERATED BY DEFAULT AS IDENTITY START WITH 106 ,
    customer_id NUMBER( 6, 0 ) NOT NULL, -- fk
    status      VARCHAR( 20 ) NOT NULL ,
    salesman_id NUMBER( 6, 0 )         , -- fk
    order_date  DATE NOT NULL   
  )
PARTITION BY RANGE (order_date) 
  INTERVAL(NUMTOYMINTERVAL(1, 'MONTH'))
    ( PARTITION INTERVAL_PARTITION_LESS_THAN_2015 VALUES LESS THAN (TO_DATE('1-1-2015', 'DD-MM-RR')),
      PARTITION INTERVAL_PARTITION_LESS_THAN_2016 VALUES LESS THAN (TO_DATE('1-1-2016', 'DD-MM-RR')),
      PARTITION INTERVAL_PARTITION_LESS_THAN_2017 VALUES LESS THAN (TO_DATE('1-7-2017', 'DD-MM-RR')),
      PARTITION INTERVAL_PARTITION_LESS_THAN_2018 VALUES LESS THAN (TO_DATE('1-1-2018', 'DD-MM-RR')) );
ALTER TABLE ORDERS_INTERVAL_PARTITION ADD (CONSTRAINT ordid_orddate_pk PRIMARY KEY (order_id, order_date));

--HASH PARTITIONS
DROP TABLESPACE tbs1 including contents;
DROP TABLESPACE tbs2 including contents;
DROP TABLESPACE tbs3 including contents;
DROP TABLESPACE tbs4 including contents;

CREATE TABLESPACE tbs1 DATAFILE SIZE 1G AUTOEXTEND ON MAXSIZE 10G;
CREATE TABLESPACE tbs2 DATAFILE SIZE 1G AUTOEXTEND ON MAXSIZE 10G;
CREATE TABLESPACE tbs3 DATAFILE SIZE 1G AUTOEXTEND ON MAXSIZE 10G;
CREATE TABLESPACE tbs4 DATAFILE SIZE 1G AUTOEXTEND ON MAXSIZE 10G;

CREATE TABLE SALES_HASH
  (s_productid  NUMBER,
   s_saledate   DATE,
   s_custid     NUMBER,
   s_totalprice NUMBER)
PARTITION BY HASH(s_productid)
( PARTITION P1 TABLESPACE tbs1
, PARTITION P2 TABLESPACE tbs2
, PARTITION P3 TABLESPACE tbs3
, PARTITION P4 TABLESPACE tbs4
);
ALTER TABLE SALES_HASH ADD (CONSTRAINT s_prod_id_pk PRIMARY KEY (s_productid, s_custid));
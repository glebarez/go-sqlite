// Copyright 2032 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drivers

import (
	"database/sql"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"modernc.org/sqlite/tpch/driver"
)

const (
	aQ1 = `select
			l_returnflag,
			l_linestatus,
			sum(l_quantity)/100. as sum_qty,
			sum(l_extendedprice)/100. as sum_base_price,
			sum(l_extendedprice*(100-l_discount))/10000. as sum_disc_price,
			sum(l_extendedprice*(100-l_discount)*(100+l_tax))/1000000. as sum_charge,
			avg(l_quantity)/100 as avg_qty,
			avg(l_extendedprice)/100 as avg_price,
			avg(l_discount)/100 as avg_disc,
			count(*) as count_order
		from
			lineitem
		where
			l_shipdate <= date('1998-12-01', printf('-%d day', ?1))
		group by
			l_returnflag,
			l_linestatus
		order by
			l_returnflag,
			l_linestatus;
`
	aQ2 = `select
			s_acctbal,
			s_name,
			n_name,
			p_partkey,
			p_mfgr,
			s_address,
			s_phone,
			s_comment
		from
			part,
			supplier,
			partsupp,
			nation,
			region
		where
			p_partkey == ps_partkey
			and s_suppkey == ps_suppkey
			and p_size == ?1
			and p_type like '%' || ?2
			and s_nationkey == n_nationkey
			and n_regionkey == r_regionkey
			and r_name == ?3
			and ps_supplycost == (
				select
					min(ps_supplycost)
				from
					partsupp, supplier,
					nation, region
				where
					p_partkey == ps_partkey
					and s_suppkey == ps_suppkey
					and s_nationkey == n_nationkey
					and n_regionkey == r_regionkey
					and r_name == ?3
			)
		order by
			s_acctbal desc,
			n_name,
			s_name,
			p_partkey
		limit 100;
`
)

func init() {
	driver.Register(newSQLite3())
}

var _ driver.SUT = (*sqlite3)(nil)

type sqlite3 struct {
	db *sql.DB
	wd string
}

func newSQLite3() *sqlite3 {
	return &sqlite3{}
}

func (b *sqlite3) Name() string            { return "sqlite3" }
func (b *sqlite3) SetWD(path string) error { b.wd = path; return nil }

func (b *sqlite3) OpenDB() (*sql.DB, error) {
	pth := filepath.Join(b.wd, "sqlite3.db")
	db, err := sql.Open(b.Name(), pth)
	if err != nil {
		return nil, err
	}

	b.db = db
	return db, nil
}

func (b *sqlite3) OpenMem() (driver.SUT, *sql.DB, error) {
	db, err := sql.Open(b.Name(), "file::memory:")
	if err != nil {
		return nil, nil, err
	}

	return &sqlite3{db: db}, db, nil
}

func (b *sqlite3) CreateTables() error {
	tx, err := b.db.Begin()
	if err != nil {
		return err
	}

	if _, err = tx.Exec(`
		create table part (
			p_partkey      int, -- SF*200,000 are populated
			p_name         string,
			p_mfgr         string,
			p_brand        string,
			p_type         string,
			p_size         int,
			p_container    string,
			p_retail_price int,
			p_comment      string
		);

		create table supplier (
			s_suppkey   int, -- SF*10,000 are populated
			s_name      string,
			s_address   string,
			s_nationkey int, -- Foreign Key to N_NATIONKEY
			s_phone     string,
			s_acctbal    int,
			s_comment   string
		);

		create table partsupp (
			ps_partkey    int, -- Foreign Key to P_PARTKEY
			ps_suppkey    int, -- Foreign Key to S_SUPPKEY
			ps_availqty   int,
			ps_supplycost int,
			ps_comment    string
		);

		create table customer (
			c_custkey    int, -- SF*150,000 are populated
			c_name       string,
			c_address    string,
			c_nationkey  int, -- Foreign Key to N_NATIONKEY
			c_phone      string,
			c_acctbal    int,
			c_mktsegment string,
			c_commnet    string
		);

		create table orders (
			o_orderkey     int, -- SF*1,500,000 are sparsely populated
			o_custkey       int, -- Foreign Key to C_CUSTKEY
			o_orderstatus   string,
			o_totalprice    int,
			o_orderdate     time,
			o_orderpriority string,
			o_clerk         string,
			o_shippriority  int,
			o_comment       string
		);

		create table lineitem (
			l_orderkey      int, -- Foreign Key to O_ORDERKEY
			l_partkey       int, -- Foreign key to P_PARTKEY, first part of the compound Foreign Key to (PS_PARTKEY, PS_SUPPKEY) with L_SUPPKEY
			l_suppkey       int, -- Foreign key to S_SUPPKEY, second part of the compound Foreign Key to (PS_PARTKEY, PS_SUPPKEY) with L_PARTKEY
			l_linenumber    int,
			l_quantity      int,
			l_extendedprice int,
			l_discount      int,
			l_tax           int,
			l_returnflag    string,
			l_linestatus    string,
			l_shipdate      time,
			l_commitdate    time,
			l_receiptdate   time,
			l_shipinstruct  string,
			l_shipmode      string,
			l_comment       string
		);

		create table nation (
			n_nationkey int, -- 25 nations are populated
			n_name      string,
			n_regionkey int, -- Foreign Key to R_REGIONKEY
			n_comment   string
		);

		create table region (
			r_regionkey int, -- 5 regions are populated
			r_name      string,
			r_comment   string
		);

		`); err != nil {
		return err
	}

	return tx.Commit()
}

func (b *sqlite3) InsertSupplier() string {
	return "insert into supplier values (?1, ?2, ?3, ?4, ?5, ?6, ?7)"
}

func (b *sqlite3) InsertPart() string {
	return "insert into part values (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9)"
}

func (b *sqlite3) InsertPartSupp() string {
	return "insert into partsupp values (?1, ?2, ?3, ?4, ?5)"
}

func (b *sqlite3) InsertCustomer() string {
	return "insert into customer values (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8)"
}

func (b *sqlite3) InsertOrders() string {
	return "insert into orders values (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9)"
}

func (b *sqlite3) InsertLineItem() string {
	return "insert into lineitem values (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13, ?14, ?15, ?16)"
}

func (b *sqlite3) InsertNation() string {
	return "insert into nation values (?1, ?2, ?3, ?4)"
}

func (b *sqlite3) InsertRegion() string {
	return "insert into region values (?1, ?2, ?3)"
}

func (b *sqlite3) QProperty() string {
	return "select * from _property"
}

func (b *sqlite3) Q1() string {
	return aQ1
}

func (b *sqlite3) Q2() string {
	return aQ2
}

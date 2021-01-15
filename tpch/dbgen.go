// Copyright 2032 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"modernc.org/mathutil"
	"modernc.org/sqlite/tpch/driver"
)

var (
	// .2.2.12 All dates must be computed using the following values:
	//
	// STARTDATE = 1992-01-01 CURRENTDATE = 1995-06-17 ENDDATE = 1998-12-31
	StartDate   = time.Date(1992, 1, 1, 0, 0, 0, 0, time.UTC)
	CurrentDate = time.Date(1995, 6, 17, 12, 12, 0, 0, time.UTC)
	EndDate     = time.Date(1998, 12, 31, 23, 59, 59, 999999999, time.UTC)

	seed, _ = mathutil.NewFCBig(big.NewInt(0), big.NewInt(math.MaxInt64), true)
	prices  []int32
	maxRecs = -1
)

func ns2time(ns int64) time.Time { return time.Unix(ns/1e9, ns%1e9).UTC() }

// 1.3 Datatype Definitions
//
// 1.3.1 The following datatype definitions apply to the list of columns of
// each table:
//
// - Identifier means that the column must be able to hold any key value
// generated for that column and be able to support at least 2,147,483,647
// unique values;
//
// Comment: A common implementation of this datatype will be an integer.
// However, for SF greater than 300 some column values will exceed the range of
// integer values supported by a 4-byte integer. A test sponsor may use some
// other datatype such as 8-byte integer, decimal or character string to
// implement the identifier datatype;
//
// - Integer means that the column must be able to exactly represent integer
// values (i.e., values in increments of 1) in the range of at least
// -2,147,483,646 to 2,147,483,647.
//
// - Decimal means that the column must be able to represent values in the
// range -9,999,999,999.99 to +9,999,999,999.99 in increments of 0.01; the
// values can be either represented exactly or interpreted to be in this range;
//
// - Big Decimal is of the Decimal datatype as defined above, with the
// additional property that it must be large enough to represent the aggregated
// values stored in temporary tables created within query variants;
//
// - Fixed text, size N means that the column must be able to hold any string
// of characters of a fixed length of N.
//
// Comment: If the string it holds is shorter than N characters, then trailing
// spaces must be stored in the database or the database must automatically pad
// with spaces upon retrieval such that a CHAR_LENGTH() function will return N.
//
// - Variable text, size N means that the column must be able to hold any
// string of characters of a variable length with a maximum length of N.
// Columns defined as "variable text, size N" may optionally be implemented as
// "fixed text, size N";
//
// - Date is a value whose external representation can be expressed as
// YYYY-MM-DD, where all characters are numeric. A date must be able to express
// any day within at least 14 consecutive years. There is no requirement
// specific to the internal representation of a date.
//
// Comment: The implementation datatype chosen by the test sponsor for a
// particular datatype definition must be applied consistently to all the
// instances of that datatype definition in the schema, except for identifier
// columns, whose datatype may be selected to satisfy database scaling
// requirements.
//
// 1.3.2 The symbol SF is used in this document to represent the scale factor
// for the database (see Clause 4: ).

type rng struct {
	r *mathutil.FCBig
}

func newRng(lo, hi int64) *rng {
	r, err := mathutil.NewFCBig(big.NewInt(lo), big.NewInt(hi), true)
	if err != nil {
		panic("internal error")
	}

	r.Seed(seed.Next().Int64())
	return &rng{r}
}

func (r *rng) n() int64 {
	return r.r.Next().Int64()
}

// 4.2.2.2 The term unique within [x] represents any one value within a set of
// x values between 1 and x, unique within the scope of rows being populated.
func uniqueWithin(x int64) *rng { return newRng(1, x) }

// 4.2.2.3 The notation random value [x .. y] represents a random value between
// x and y inclusively, with a mean of (x+y)/2, and with the same number of
// digits of precision as shown. For example, [0.01 .. 100.00] has 10,000
// unique values, whereas [1..100] has only 100 unique values.
func (r *rng) randomValue(min, max int64) int64 {
	return min + r.n()%(max-min+1)
}

// 4.2.2.7 The notation random v-string [min, max] represents a string
// comprised of randomly generated alphanumeric characters within a character
// set of at least 64 symbols. The length of the string is a random value
// between min and max inclusive.
func (r *rng) vString(min, max int) string {
	l := min + int(r.n())%(max-min+1)
	b := make([]byte, l)
	for i := range b {
		b[i] = '0' + byte(r.n()%64)
	}
	return string(b)
}

// 4.2.2.9 The term phone number represents a string of numeric characters
// separated by hyphens and generated as follows:
//
// Let i be an index into the list of strings Nations (i.e., ALGERIA is 0,
// ARGENTINA is 1, etc., see Clause 4.2.3),
//
// Let country_code be the sub-string representation of the number (i + 10),
//
// Let local_number1 be random [100 .. 999],
//
// Let local_number2 be random [100 .. 999],
//
// Let local_number3 be random [1000 .. 9999],
//
// The phone number string is obtained by concatenating the following
// sub-strings:
//
// country_code, "-", local_number1, "-", local_number2, "-", local_number3
func (r *rng) phoneNumber(i int) string {
	return fmt.Sprintf("%v-%v-%v-%v", i+10, 100+r.n()%900, 100+r.n()%900, 1000+r.n()%9000)
}

// 4.2.2.10 The term text string[min, max] represents a substring of a 300 MB
// string populated according to the pseudo text grammar defined in Clause
// 4.2.2.14. The length of the substring is a random number between min and max
// inclusive. The substring offset is randomly chosen.
func (r *rng) textString(min, max int64) string {
	off := r.n() % (int64(len(pseudotext)) - max)
	l := min + r.n()%(max-min+1)
	return string(pseudotext[off : off+l])
}

var (
	types1 = []string{
		"STANDARD",
		"SMALL",
		"MEDIUM",
		"LARGE",
		"ECONOMY",
		"PROMO",
	}
	types2 = []string{
		"ANODIZED",
		"BURNISHED",
		"PLATED",
		"POLISHED",
		"BRUSHED",
	}
	types3 = []string{
		"TIN",
		"NICKEL",
		"BRASS",
		"STELL",
		"COPPER",
	}
)

// 4.2.2.13

func (r *rng) types() string {
	return types1[int(r.n())%len(types1)] + " " + types2[int(r.n())%len(types2)] + " " + types3[int(r.n())%len(types3)]
}

var (
	containers1 = []string{
		"SM",
		"LG",
		"MED",
		"JUMBO",
		"WRAP",
	}
	containers2 = []string{
		"CASE",
		"BOX",
		"BAG",
		"JAR",
		"PKG",
		"PACK",
		"CAN",
		"DRUM",
	}
)

func (r *rng) containers() string {
	return containers1[int(r.n())%len(containers1)] + " " + containers2[int(r.n())%len(containers2)]
}

var segments1 = []string{
	"AUTOMOBILE",
	"BUILDING",
	"FURNITURE",
	"MACHINERY",
	"HOUSEHOLD",
}

func (r *rng) segments() string {
	return segments1[int(r.n())%len(segments1)]
}

var priorities1 = []string{
	"1-URGENT",
	"2-HIGH",
	"3-MEDIUM",
	"4-NOT SPECIFIED",
	"5-LOW",
}

func (r *rng) priorities() string {
	return priorities1[int(r.n())%len(priorities1)]
}

var instructions1 = []string{
	"DELIVER IN PERSON",
	"COLLECT COD",
	"NONE",
	"TAKE BACK RETURN",
}

func (r *rng) instructions() string {
	return instructions1[int(r.n())%len(instructions1)]
}

var modes1 = []string{
	"REG AIR",
	"AIR",
	"RAIL",
	"SHIP",
	"TRUCK",
	"MAIL",
	"FOB",
}

func (r *rng) modes() string {
	return modes1[int(r.n())%len(modes1)]
}

var nouns1 = []string{
	"foxes",
	"ideas",
	"theodolites",
	"pinto",
	"beans",
	"instructions",
	"dependencies",
	"excuses",
	"platelets",
	"asymptotes",
	"courts",
	"dolphins",
	"multipliers",
	"sauternes",
	"warthogs",
	"frets",
	"dinos",
	"attainments",
	"somas",
	"Tiresias'",
	"patterns",
	"forges",
	"braids",
	"hockey",
	"players",
	"frays",
	"warhorses",
	"dugouts",
	"notornis",
	"epitaphs",
	"pearls",
	"tithes",
	"waters",
	"orbits",
	"gifts",
	"sheaves",
	"depths",
	"sentiments",
	"decoys",
	"realms",
	"pains",
	"grouches",
	"escapades",
}

func (r *rng) nouns() string {
	return nouns1[int(r.n())%len(nouns1)]
}

var verbs1 = []string{
	"sleep",
	"wake",
	"are",
	"cajole",
	"haggle",
	"nag",
	"use",
	"boost",
	"affix",
	"detect",
	"integrate",
	"maintain",
	"nod",
	"was",
	"lose",
	"sublate",
	"solve",
	"thrash",
	"promise",
	"engage",
	"hinder",
	"print",
	"x-ray",
	"breach",
	"eat",
	"grow",
	"impress",
	"mold",
	"poach",
	"serve",
	"run",
	"dazzle",
	"snooze",
	"doze",
	"unwind",
	"kindle",
	"play",
	"hang",
	"believe",
	"doubt",
}

func (r *rng) verbs() string {
	return verbs1[int(r.n())%len(verbs1)]
}

var adjectives1 = []string{
	"furious",
	"sly",
	"careful",
	"blithe",
	"quick",
	"fluffy",
	"slow",
	"quiet",
	"ruthless",
	"thin",
	"close",
	"dogged",
	"daring",
	"brave",
	"stealthy",
	"permanent",
	"enticing",
	"idle",
	"busy",
	"regular",
	"final",
	"ironic",
	"even",
	"bold",
	"silent",
}

func (r *rng) adjectives() string {
	return adjectives1[int(r.n())%len(adjectives1)]
}

var adverbs1 = []string{
	"sometimes",
	"always",
	"never",
	"furiously",
	"slyly",
	"carefully",
	"blithely",
	"quickly",
	"fluffily",
	"slowly",
	"quietly",
	"ruthlessly",
	"thinly",
	"closely",
	"doggedly",
	"daringly",
	"bravely",
	"stealthily",
	"permanently",
	"enticingly",
	"idly",
	"busily",
	"regularly",
	"finally",
	"ironically",
	"evenly",
	"boldly",
	"silently",
}

func (r *rng) adverbs() string {
	return adverbs1[int(r.n())%len(adverbs1)]
}

var prepositions1 = []string{
	"about",
	"above",
	"according to",
	"across",
	"after",
	"against",
	"along",
	"alongside of",
	"among",
	"around",
	"at",
	"atop",
	"before",
	"behind",
	"beneath",
	"beside",
	"besides",
	"between",
	"beyond",
	"by",
	"despite",
	"during",
	"except",
	"for",
	"from",
	"in place of",
	"inside",
	"instead of",
	"into",
	"near",
	"of",
	"on",
	"outside",
	"over",
	"past",
	"since",
	"through",
	"throughout",
	"to",
	"toward",
	"under",
	"until",
	"up",
	"upon",
	"without",
	"with",
	"within",
}

func (r *rng) prepositions() string {
	return prepositions1[int(r.n())%len(prepositions1)]
}

var auxiliaries1 = []string{
	"do",
	"may",
	"might",
	"shall",
	"will",
	"would",
	"can",
	"could",
	"should",
	"ought to",
	"must",
	"will have to",
	"shall have to",
	"could have to",
	"should have to",
	"must have to",
	"need to",
	"try to",
}

func (r *rng) auxiliaries() string {
	return auxiliaries1[int(r.n())%len(auxiliaries1)]
}

var terminators1 = []string{
	".",
	";",
	"!",
	":",
	"?",
	"--",
}

func (r *rng) terminators() string {
	return terminators1[int(r.n())%len(terminators1)]
}

var pseudotext []byte

func genPseudotext() (err error) {
	pth := filepath.Join("testdata", "pseudotext")
	if _, err = os.Stat(pth); err == nil {
		return fmt.Errorf("file already exists: %s", pth)
	}

	if !os.IsNotExist(err) {
		return err
	}

	if err = os.MkdirAll("testdata", 0766); err != nil {
		return err
	}

	f, err := os.Create(pth)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	w := bufio.NewWriter(f)

	defer func() {
		if ferr := w.Flush(); ferr != nil && err == nil {
			err = ferr
		}
	}()

	const sz = 300 * 1e6
	r := newRng(0, math.MaxInt64)

	nounPhrase := func() string {
		switch r.n() % 4 {
		case 0: // noun phrase:<noun>
			return r.nouns()
		case 1: // |<adjective> <noun>
			return r.adjectives() + " " + r.nouns()
		case 2: // |<adjective>, <adjective> <noun>
			return r.adjectives() + ", " + r.adjectives() + " " + r.nouns()
		case 3: // |<adverb> <adjective> <noun>
			return r.adverbs() + " " + r.adjectives() + " " + r.nouns()
		}
		panic("internal error")
	}

	verbPhrase := func() string {
		switch r.n() % 4 {
		case 0: // verb phrase:<verb>
			return r.verbs()
		case 1: // |<auxiliary> <verb>
			return r.auxiliaries() + " " + r.verbs()
		case 2: // |<verb> <adverb>
			return r.verbs() + " " + r.adverbs()
		case 3: // |<auxiliary> <verb> <adverb>
			return r.auxiliaries() + " " + r.verbs() + " " + r.adverbs()
		}
		panic("internal error")
	}

	prepositionalPhrase := func() string {
		// prepositional phrase: <preposition> the <noun phrase>
		return r.prepositions() + " the " + nounPhrase()
	}

	sentence := func() string {
		switch r.n() % 5 {
		case 0: // sentence:<noun phrase> <verb phrase> <terminator>
			return nounPhrase() + " " + verbPhrase() + r.terminators()
		case 1: // |<noun phrase> <verb phrase> <prepositional phrase> <terminator>
			return nounPhrase() + " " + verbPhrase() + " " + prepositionalPhrase() + r.terminators()
		case 2: // |<noun phrase> <verb phrase> <noun phrase> <terminator>
			return nounPhrase() + " " + verbPhrase() + " " + nounPhrase() + r.terminators()
		case 3: // |<noun phrase> <prepositional phrase> <verb phrase> <noun phrase> <terminator>
			return nounPhrase() + " " + prepositionalPhrase() + " " + verbPhrase() + " " + nounPhrase() + r.terminators()
		case 4: // |<noun phrase> <prepositional phrase> <verb phrase> <prepositional phrase> <terminator>
			return nounPhrase() + " " + prepositionalPhrase() + " " + verbPhrase() + " " + prepositionalPhrase() + r.terminators()
		}
		panic("internal error")
	}

	n := 0
	for n < sz {
		s := sentence() + " "
		if _, err = w.WriteString(s); err != nil {
			return err
		}

		n += len(s)
	}
	return nil
}

func pthForSUT(sut driver.SUT, sf int) string {
	return filepath.Join("testdata", sut.Name(), "sf"+strconv.Itoa(sf))
}

func dbGen(sut driver.SUT, sf int) (err error) {
	if pseudotext, err = ioutil.ReadFile(filepath.Join("testdata", "pseudotext")); err != nil {
		return fmt.Errorf("Run this program with -pseudotext: %v", err)
	}

	pth := pthForSUT(sut, sf)
	if err = os.MkdirAll(pth, 0766); err != nil {
		return err
	}

	if err = sut.SetWD(pth); err != nil {
		return err
	}

	db, err := sut.OpenDB()
	if err != nil {
		return err
	}

	defer func() {
		if cerr := db.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if err = sut.CreateTables(); err != nil {
		return err
	}

	if err = genSupplier(db, sf, sut); err != nil {
		return err
	}

	if err = genPartAndPartSupp(db, sf, sut); err != nil {
		return err
	}

	if err = genCustomerAndOrders(db, sf, sut); err != nil {
		return err
	}

	if err = genNation(db, sf, sut); err != nil {
		return err
	}

	return genRegion(db, sf, sut)
}

func genSupplier(db *sql.DB, sf int, sut driver.SUT) (err error) {
	recs := 10000
	if n := maxRecs; n >= 0 {
		recs = n
	}

	keyrng := uniqueWithin(int64(sf) * int64(recs))
	rng5 := uniqueWithin(int64(sf) * int64(recs))
	sf5rows := make(map[int64]bool)
	for i := 0; i < sf*5; i++ {
		sf5rows[rng5.n()] = true
		sf5rows[rng5.n()] = false
	}
	rng := newRng(0, math.MaxInt64)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(sut.InsertSupplier())
	if err != nil {
		return err
	}

	// SF * 10,000 rows in the SUPPLIER table with:
	// S_SUPPKEY unique within [SF * 10,000].
	// S_NAME text appended with minimum 9 digits with leading zeros ["Supplie#r", S_SUPPKEY].
	// S_ADDRESS random v-string[10,40].
	// S_NATIONKEY random value [0 .. 24].
	// S_PHONE generated according to Clause 4.2.2.9.
	// S_ACCTBAL random value [-999.99 .. 9,999.99].
	// S_COMMENT text string [25,100].
	//	SF * 5 rows are randomly selected to hold at a random position
	//	a string matching "Customer%Complaints". Another SF * 5 rows
	//	are randomly selected to hold at a random position a string
	//	matching "Customer%Recommends", where % is a wildcard that
	//	denotes zero or more characters.
	for i := 0; i < sf*recs; i++ {
		sSuppKey := keyrng.n()
		nk := int(rng.n() % 25)
		sComment := rng.textString(25, 100)
		if b, ok := sf5rows[sSuppKey]; ok {
			s := "Complaints"
			if b {
				s = "Recommends"
			}
			s = "Customer" + rng.vString(0, 4) + s
			off := int(rng.randomValue(0, int64(len(sComment)-len(s))))
			sComment = sComment[:off] + s + sComment[off+len(s):]
		}
		if _, err := stmt.Exec(
			sSuppKey,
			fmt.Sprintf("Supplier#%09d", sSuppKey),
			rng.vString(10, 40),
			nk,
			rng.phoneNumber(nk),
			rng.randomValue(-99999, 999999),
			sComment,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

var pnames1 = []string{
	"almond", "antique", "aquamarine", "azure", "beige", "bisque", "black", "blanched", "blue",
	"blush", "brown", "burlywood", "burnished", "chartreuse", "chiffon", "chocolate", "coral",
	"cornflower", "cornsilk", "cream", "cyan", "dark", "deep", "dim", "dodger", "drab", "firebrick",
	"floral", "forest", "frosted", "gainsboro", "ghost", "goldenrod", "green", "grey", "honeydew",
	"hot", "indian", "ivory", "khaki", "lace", "lavender", "lawn", "lemon", "light", "lime", "linen",
	"magenta", "maroon", "medium", "metallic", "midnight", "mint", "misty", "moccasin", "navajo",
	"navy", "olive", "orange", "orchid", "pale", "papaya", "peach", "peru", "pink", "plum", "powder",
	"puff", "purple", "red", "rose", "rosy", "royal", "saddle", "salmon", "sandy", "seashell", "sienna",
	"sky", "slate", "smoke", "snow", "spring", "steel", "tan", "thistle", "tomato", "turquoise", "violet",
	"wheat", "white", "yellow",
}

func genPartAndPartSupp(db *sql.DB, sf int, sut driver.SUT) (err error) {
	recs := 200000
	if n := maxRecs; n >= 0 {
		recs = n
	}

	prices = make([]int32, sf*recs)
	a := make([]string, 5)
	var tx *sql.Tx
	var stmt, stmt2 *sql.Stmt

	// SF * 200,000 rows in the PART table with:
	// P_PARTKEY unique within [SF * 200,000].
	// P_NAME generated by concatenating five unique randomly selected strings from the following list,
	// separated by a single space:
	//	{"almond", "antique", "aquamarine", "azure", "beige", "bisque", "black", "blanched", "blue",
	//	"blush", "brown", "burlywood", "burnished", "chartreuse", "chiffon", "chocolate", "coral",
	//	"cornflower", "cornsilk", "cream", "cyan", "dark", "deep", "dim", "dodger", "drab", "firebrick",
	//	"floral", "forest", "frosted", "gainsboro", "ghost", "goldenrod", "green", "grey", "honeydew",
	//	"hot", "indian", "ivory", "khaki", "lace", "lavender", "lawn", "lemon", "light", "lime", "linen",
	//	"magenta", "maroon", "medium", "metallic", "midnight", "mint", "misty", "moccasin", "navajo",
	//	"navy", "olive", "orange", "orchid", "pale", "papaya", "peach", "peru", "pink", "plum", "powder",
	//	"puff", "purple", "red", "rose", "rosy", "royal", "saddle", "salmon", "sandy", "seashell", "sienna",
	//	"sky", "slate", "smoke", "snow", "spring", "steel", "tan", "thistle", "tomato", "turquoise", "violet",
	//	"wheat", "white", "yellow"}.
	// P_MFGR text appended with digit ["Manufacturer#",M], where M = random value [1,5].
	// P_BRAND text appended with digits ["Brand#",MN], where N = random value [1,5] and M is defined
	//	while generating P_MFGR.
	// P_TYPE random string [Types].
	// P_SIZE random value [1 .. 50].
	// P_CONTAINER random string [Containers].
	// P_RETAILPRICE = (90000 + ((P_PARTKEY/10) modulo 20001 ) + 100 * (P_PARTKEY modulo 1000))/100
	// P_COMMENT text string [5,22].
	keyrng := uniqueWithin(int64(sf) * int64(recs))
	rng := newRng(0, math.MaxInt64)
	for i := 0; i < sf*recs; i++ {
		if i%1000 == 0 {
			if i != 0 {
				if err = tx.Commit(); err != nil {
					return err
				}
			}

			if tx, err = db.Begin(); err != nil {
				return err
			}

			if stmt, err = tx.Prepare(sut.InsertPart()); err != nil {
				return err
			}

			if stmt2, err = tx.Prepare(sut.InsertPartSupp()); err != nil {
				return err
			}
		}
		pPartKey := keyrng.n()
		a = a[:0]
		for len(a) < 5 {
		again:
			s := pnames1[rng.n()%int64(len(pnames1))]
			for _, v := range a {
				if s == v {
					goto again
				}
			}

			a = append(a, s)
		}
		m := rng.randomValue(1, 5)
		pRetailPrice := 90000 + ((pPartKey / 10) % 20001) + 100*(pPartKey%1000)
		prices[pPartKey-1] = int32(pRetailPrice)
		if _, err := stmt.Exec(
			pPartKey,
			strings.Join(a, " "),
			fmt.Sprintf("Manufacturer#%d", m),
			fmt.Sprintf("Brand#%d%d", m, rng.randomValue(1, 5)),
			rng.types(),
			rng.randomValue(1, 50),
			rng.containers(),
			pRetailPrice,
			rng.textString(5, 22),
		); err != nil {
			return err
		}

		// For each row in the PART table, four rows in PartSupp table with:
		// PS_PARTKEY = P_PARTKEY.
		// PS_SUPPKEY = (ps_partkey + (i * (( S/4 ) + (int)(ps_partkey-1 )/S)))) modulo S + 1
		//	where i is the ith supplier within [0 .. 3] and S = SF * 10,000.
		// PS_AVAILQTY random value [1 .. 9,999].
		// PS_SUPPLYCOST random value [1.00 .. 1,000.00].
		// PS_COMMENT text string [49,198].
		s := int64(sf) * 10000
		psPartKey := pPartKey
		for i := 0; i < 4; i++ {
			if _, err := stmt2.Exec(
				psPartKey,
				(psPartKey+(int64(i)*((s/4)+(psPartKey-1)/s)))%(s+1),
				rng.randomValue(1, 9999),
				rng.randomValue(100, 100000),
				rng.textString(49, 198),
			); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func genCustomerAndOrders(db *sql.DB, sf int, sut driver.SUT) (err error) {
	recs := 150000
	if n := maxRecs; n >= 0 {
		recs = n
	}

	var tx *sql.Tx
	var stmt, stmt2, stmt3 *sql.Stmt
	s := int64(sf) * 10000

	minDate := StartDate.UnixNano()
	maxDate := EndDate.UnixNano() - 151*24*int64(time.Hour)

	// SF * 150,000 rows in CUSTOMER table with:
	// C_CUSTKEY unique within [SF * 150,000].
	// C_NAME text appended with minimum 9 digits with leading zeros ["Customer#", C_CUSTKEY].
	// C_ADDRESS random v-string [10,40].
	// C_NATIONKEY random value [0 .. 24].
	// C_PHONE generated according to Clause 4.2.2.9.
	// C_ACCTBAL random value [-999.99 .. 9,999.99].
	// C_MKTSEGMENT random string [Segments].
	// C_COMMENT text string [29,116].
	keyrng := uniqueWithin(int64(sf) * int64(recs))
	keyrng2 := uniqueWithin(int64(sf) * 10 * int64(recs))
	rng := newRng(0, math.MaxInt64)
	for i := 0; i < sf*recs; i++ {
		if i%1000 == 0 {
			if i != 0 {
				if err = tx.Commit(); err != nil {
					return err
				}
			}

			if tx, err = db.Begin(); err != nil {
				return err
			}

			if stmt, err = tx.Prepare(sut.InsertCustomer()); err != nil {
				return err
			}

			if stmt2, err = tx.Prepare(sut.InsertOrders()); err != nil {
				return err
			}

			if stmt3, err = tx.Prepare(sut.InsertLineItem()); err != nil {
				return err
			}
		}
		cCustKey := keyrng.n()
		nk := rng.randomValue(0, 24)
		if _, err := stmt.Exec(
			cCustKey,
			fmt.Sprintf("Customer#%09d", cCustKey),
			rng.vString(10, 40),
			nk,
			rng.phoneNumber(int(nk)),
			rng.randomValue(-99999, 999999),
			rng.segments(),
			rng.textString(29, 116),
		); err != nil {
			return err
		}

		// For each row in the CUSTOMER table, ten rows in the ORDERS
		// table with: O_ORDERKEY unique within [SF * 1,500,000 * 4].
		//
		// Comment: The ORDERS and LINEITEM tables are sparsely
		// populated by generating a key value that causes the first 8
		// keys of each 32 to be populated, yielding a 25% use of the
		// key range. Test sponsors must not take advantage of this
		// aspect of the benchmark. For example, horizontally
		// partitioning the test database onto different devices in
		// order to place unused areas onto separate peripherals is
		// prohibited.
		//
		// O_CUSTKEY = random value [1 .. (SF * 150,000)].
		//	The generation of this random value must be such that
		//	O_CUSTKEY modulo 3 is not zero.
		//
		//	Comment: Orders are not present for all customers. Every
		//	third customer (in C_CUSTKEY order) is not assigned any
		//	order.
		//
		// O_ORDERSTATUS set to the following value:
		//	"F" if all lineitems of this order have L_LINESTATUS set to "F".
		//	"O" if all lineitems of this order have L_LINESTATUS set to "O".
		//	"P" otherwise.
		// O_TOTALPRICE computed as:
		//	sum (L_EXTENDEDPRICE * (1+L_TAX) * (1-L_DISCOUNT)) for all LINEITEM of this order.
		// O_ORDERDATE uniformly distributed between STARTDATE and (ENDDATE - 151 days).
		// O_ORDERPRIORITY random string [Priorities].
		// O_CLERK text appended with minimum 9 digits with leading zeros ["Clerk#", C] where C = random value [000000001 .. (SF * 1000)].
		// O_SHIPPRIORITY set to 0.
		// O_COMMENT text string [19,78].

		for i := 0; i < 10; i++ {
			var oCustKey int64
			for {
				oCustKey = rng.randomValue(1, int64(sf)*int64(recs))
				if oCustKey%3 != 0 {
					break
				}
			}
			oOrderKey := keyrng2.n() - 1                    // Zero base.
			oOrderKey = oOrderKey/8*32 + oOrderKey%8 + 1    // 1 based, sparseness as specified above.
			oOrderDate := rng.randomValue(minDate, maxDate) // unix ns
			oOrderStatus := "X"
			var oTotalPrice int64

			{
				// For each row in the ORDERS table, a random number of rows within [1 .. 7] in the LINEITEM table with:
				// L_ORDERKEY = O_ORDERKEY.
				// L_PARTKEY random value [1 .. (SF * 200,000)].
				// L_SUPPKEY = (L_PARTKEY + (i * (( S/4 ) + (int)(L_partkey-1 )/S)))) modulo S + 1
				//	where i is the corresponding supplier within [0 .. 3] and S = SF * 10,000.
				// L_LINENUMBER unique within [7].
				// L_QUANTITY random value [1 .. 50].
				// L_EXTENDEDPRICE = L_QUANTITY * P_RETAILPRICE
				//	Where P_RETAILPRICE is from the part with P_PARTKEY = L_PARTKEY.
				// L_DISCOUNT random value [0.00 .. 0.10].
				// L_TAX random value [0.00 .. 0.08].
				// L_RETURNFLAG set to a value selected as follows:
				//	If L_RECEIPTDATE <= CURRENTDATE
				//	then either "R" or "A" is selected at random
				//	else "N" is selected.
				// L_LINESTATUS set the following value:
				//	"O" if L_SHIPDATE > CURRENTDATE
				//	"F" otherwise.
				// L_SHIPDATE = O_ORDERDATE + random value [1 .. 121].
				// L_COMMITDATE = O_ORDERDATE + random value [30 .. 90].
				// L_RECEIPTDATE = L_SHIPDATE + random value [1 .. 30].
				// L_SHIPINSTRUCT random string [Instructions].
				// L_SHIPMODE random string [Modes].
				// L_COMMENT text string [10,43].
				n := int(rng.randomValue(1, 7))
				lRng := uniqueWithin(7)
				qty := rng.randomValue(100, 5000)
				lShipDate := ns2time(oOrderDate + rng.randomValue(1, 121)*24*int64(time.Hour))
				lCommitDate := ns2time(oOrderDate + rng.randomValue(30, 90)*24*int64(time.Hour))
				lReceiptDate := ns2time(oOrderDate + rng.randomValue(1, 30)*24*int64(time.Hour))
				lReturnFlag := "X"
				switch {
				case lReceiptDate.Before(CurrentDate) || lReceiptDate.Equal(CurrentDate):
					if rng.n()&1 == 0 {
						lReturnFlag = "R"
						break
					}

					lReturnFlag = "A"
				default:
					lReturnFlag = "N"
				}
				lLineStatus := "F"
				if lShipDate.After(CurrentDate) {
					lLineStatus = "O"
				}
				switch {
				case oOrderStatus == "X":
					oOrderStatus = lLineStatus
				case oOrderStatus != lLineStatus:
					oOrderStatus = "P"
				}
				for i := 0; i < n; i++ {
					lPartKey := rng.randomValue(1, int64(len(prices)))
					pRetailPrice := int64(prices[lPartKey-1])
					lExtendedPrice := qty * pRetailPrice / 100
					lTax := rng.randomValue(0, 8)
					lDiscount := rng.randomValue(0, 10)
					oTotalPrice += lExtendedPrice * (100 + lTax) * (100 - lDiscount) / 100 / 100
					if _, err := stmt3.Exec(
						oOrderKey,
						lPartKey,
						(lPartKey+(int64(i)*(s/4+(lPartKey-1)/s)))%(s+1),
						lRng.n(),
						qty,
						lExtendedPrice,
						lDiscount,
						lTax,
						lReturnFlag,
						lLineStatus,
						lShipDate,
						lCommitDate,
						lReceiptDate,
						rng.instructions(),
						rng.modes(),
						rng.textString(10, 43),
					); err != nil {
						return err
					}
				}
			}

			if _, err := stmt2.Exec(
				oOrderKey,
				oCustKey,
				oOrderStatus,
				oTotalPrice,
				ns2time(oOrderDate/1e9),
				rng.priorities(),
				fmt.Sprintf("Clerk#%09d", rng.randomValue(1, int64(sf)*1000)),
				0,
				rng.textString(19, 78),
			); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

var nations = []struct {
	name      string
	regionKey int
}{
	{"ALGERIA", 0},
	{"ARGENTINA", 1},
	{"BRAZIL", 1},
	{"CANADA", 1},
	{"EGYPT", 4},
	{"ETHIOPIA", 0},
	{"FRANCE", 3},
	{"GERMANY", 3},
	{"INDIA", 2},
	{"INDONESIA", 2},
	{"IRAN", 4},
	{"IRAQ", 4},
	{"JAPAN", 2},
	{"JORDAN", 4},
	{"KENYA", 0},
	{"MOROCCO", 0},
	{"MOZAMBIQUE", 0},
	{"PERU", 1},
	{"CHINA", 2},
	{"ROMANIA", 3},
	{"SAUDI ARABIA", 4},
	{"VIETNAM", 2},
	{"RUSSIA", 3},
	{"UNITED KINGDOM", 3},
	{"UNITED STATES", 1},
}

func genNation(db *sql.DB, sf int, sut driver.SUT) (err error) {
	rng := newRng(0, math.MaxInt64)
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(sut.InsertNation())
	if err != nil {
		return err
	}

	// 25 rows in the NATION table with:
	// N_NATIONKEY unique value between 0 and 24.
	// N_NAME string from the following series of (N_NATIONKEY, N_NAME, N_REGIONKEY).
	// (0, ALGERIA, 0);(1, ARGENTINA, 1);(2, BRAZIL, 1);
	// (3, CANADA, 1);(4, EGYPT, 4);(5, ETHIOPIA, 0);
	// (6, FRANCE, 3);(7, GERMANY, 3);(8, INDIA, 2);
	// (9, INDONESIA, 2);(10, IRAN, 4);(11, IRAQ, 4);
	// (12, JAPAN, 2);(13, JORDAN, 4);(14, KENYA, 0);
	// (15, MOROCCO, 0);(16, MOZAMBIQUE, 0);(17, PERU, 1);
	// (18, CHINA, 2);(19, ROMANIA, 3);(20, SAUDI ARABIA, 4);
	// (21, VIETNAM, 2);(22, RUSSIA, 3);(23, UNITED KINGDOM, 3);
	// (24, UNITED STATES, 1)
	// N_REGIONKEY is taken from the series above.
	// N_COMMENT text string [31,114].
	for i, v := range nations {
		if _, err := stmt.Exec(
			int64(i),
			v.name,
			int64(v.regionKey),
			rng.textString(31, 114),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

var regions1 = []string{
	"AFRICA",
	"AMERICA",
	"ASIA",
	"EUROPE",
	"MIDDLE EAST",
}

func (r *rng) regions() string {
	return regions1[int(r.n())%len(regions1)]
}

func genRegion(db *sql.DB, sf int, sut driver.SUT) (err error) {
	rng := newRng(0, math.MaxInt64)
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(sut.InsertRegion())
	if err != nil {
		return err
	}

	// 5 rows in the REGION table with:
	// R_REGIONKEY unique value between 0 and 4.
	// R_NAME string from the following series of (R_REGIONKEY, R_NAME).
	// (0, AFRICA);(1, AMERICA);
	// (2, ASIA);
	// (3, EUROPE);(4, MIDDLE EAST)
	// R_COMMENT text string [31,115].
	for i, v := range regions1 {
		if _, err := stmt.Exec(
			int64(i),
			v,
			rng.textString(31, 115),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

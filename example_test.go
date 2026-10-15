package territory_test

import (
	"fmt"

	territory "github.com/caturseptian/go-indonesia-territory"
)

func ExampleProvinceByCode() {
	p, err := territory.ProvinceByCode("31")
	if err != nil {
		panic(err)
	}
	fmt.Println(p.Code, p.Name)
	// Output: 31 Daerah Khusus Ibukota Jakarta
}

func ExampleParse() {
	c, err := territory.Parse("31.71.01.1001")
	fmt.Println(c.Level, c.Parent, err)
	_, err = territory.Parse("3171")
	fmt.Println(err)
	// Output:
	// village 31.71.01 <nil>
	// territory: kode tidak berformat Kepmendagri: "3171"
}

func ExamplePathOf() {
	p, err := territory.PathOf("31.71.01.1001")
	if err != nil {
		panic(err)
	}
	fmt.Println(p.Province.Name, "/", p.Regency.Name, "/", p.District.Name, "/", p.Village.Name)
	// Output: Daerah Khusus Ibukota Jakarta / Kota Administrasi Jakarta Pusat / Gambir / Gambir
}

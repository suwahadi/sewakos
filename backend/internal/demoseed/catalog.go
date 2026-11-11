package demoseed

type photoKind int

const (
	kindExterior photoKind = iota
	kindCommon
	kindBathroom
	kindBedroom
)

var photoCaptions = map[photoKind]string{
	kindExterior: "Tampak depan",
	kindCommon:   "Area bersama",
	kindBathroom: "Kamar mandi",
}

// Unsplash photo IDs (Unsplash License: free to use, no attribution required), checked by eye for each category.
var photoSources = map[photoKind][]string{
	kindExterior: {
		"1757970326337-95d7cca56fa1", "1755735340764-3b077cab0c5c", "1757221254460-e1139728b25c",
		"1545324418-cc1a3fa10c00", "1600585154340-be6161a56a0c", "1600566753190-17f0baa2a6c3",
	},
	kindCommon: {
		"1522708323590-d24dbb6b0267", "1502672260266-1c1ef2d93688", "1493809842364-78817add7ffb",
		"1484154218962-a197022b5858", "1586023492125-27b2c045efd7", "1513694203232-719a280e022f",
		"1560185007-cde436f6a4d0", "1502005229762-cf1b2da7c5d6", "1617806118233-18e1de247200",
		"1507089947368-19c1da9775ae",
	},
	kindBathroom: {
		"1584622650111-993a426fbf0a", "1552321554-5fefe8c9ef14", "1620626011761-996317b8d101",
	},
	kindBedroom: {
		"1505693416388-ac5ce068fe85", "1540518614846-7eded433c457", "1631049307264-da0ec9d70304",
		"1616594039964-ae9021a400a0", "1595526114035-0d45ed16cfbf", "1615874959474-d609969a20ed",
		"1618773928121-c32242e63f39", "1522771739844-6a9f6d5f14af",
	},
}

type demoOwner struct {
	name, phone, bank, account string
}

var demoOwners = []demoOwner{
	{"Sri Wahyuni", "081100000001", "BCA", "8730012345"},
	{"Agus Santoso", "081100000002", "Mandiri", "1370004567812"},
	{"Rina Kartika", "081100000003", "BNI", "0456123789"},
	{"Dedi Kurniawan", "081100000004", "BRI", "002101234567509"},
	{"Lestari Handayani", "081100000005", "BCA", "5410098765"},
}

type areaDef struct {
	name, slug, kind string
	isTop            bool
	lat, lng         float64
}

type spot struct {
	landmark      *areaDef
	lat, lng      float64
	district      string
	neighborhoods []string
	streets       []string
}

type cityDef struct {
	area               areaDef
	priceMin, priceMax int64
	spots              []spot
}

var cities = []cityDef{
	{
		area: areaDef{"Jakarta Selatan", "jakarta-selatan", "city", true, -6.2615, 106.8106}, priceMin: 1_800_000, priceMax: 4_200_000,
		spots: []spot{
			{&areaDef{"MRT Blok M", "mrt-blok-m", "mrt", true, -6.2443, 106.7983}, -6.2443, 106.7983, "Kebayoran Baru",
				[]string{"Blok M", "Melawai", "Gandaria"}, []string{"Jl. Melawai Raya", "Jl. Bulungan", "Jl. Gandaria Tengah"}},
			{&areaDef{"MRT Lebak Bulus", "mrt-lebak-bulus", "mrt", false, -6.2894, 106.7745}, -6.2894, 106.7745, "Cilandak",
				[]string{"Lebak Bulus", "Fatmawati", "Cilandak"}, []string{"Jl. RS Fatmawati", "Jl. Lebak Bulus Raya", "Jl. Cilandak KKO"}},
			{nil, -6.2266, 106.8486, "Tebet", []string{"Tebet", "Menteng Dalam", "Kasablanka"}, []string{"Jl. Tebet Raya", "Jl. Tebet Barat Dalam", "Jl. Casablanca"}},
		},
	},
	{
		area: areaDef{"Jakarta Pusat", "jakarta-pusat", "city", true, -6.1865, 106.8341}, priceMin: 1_700_000, priceMax: 3_900_000,
		spots: []spot{
			{&areaDef{"MRT Bundaran HI", "mrt-bundaran-hi", "mrt", true, -6.1913, 106.8230}, -6.1913, 106.8230, "Menteng",
				[]string{"Kebon Kacang", "Gondangdia", "Cikini"}, []string{"Jl. Kebon Kacang Raya", "Jl. Cikini Raya", "Jl. Wahid Hasyim"}},
			{nil, -6.1745, 106.8227, "Tanah Abang", []string{"Petamburan", "Bendungan Hilir", "Karet"}, []string{"Jl. KH Mas Mansyur", "Jl. Bendungan Hilir", "Jl. Petamburan"}},
		},
	},
	{
		area: areaDef{"Kota Depok", "kota-depok", "city", true, -6.4025, 106.7942}, priceMin: 1_000_000, priceMax: 2_600_000,
		spots: []spot{
			{&areaDef{"Universitas Indonesia", "universitas-indonesia", "campus", true, -6.3608, 106.8272}, -6.3608, 106.8272, "Beji",
				[]string{"Margonda", "Kukusan", "Kutek", "Pondok Cina"}, []string{"Jl. Margonda Raya", "Jl. Kukusan Teknik", "Jl. Haji Amat", "Jl. Sawo"}},
			{nil, -6.4025, 106.7942, "Pancoran Mas", []string{"Depok Lama", "Mampang", "Rangkapan Jaya"}, []string{"Jl. Nusantara Raya", "Jl. Arif Rahman Hakim", "Jl. Raya Sawangan"}},
		},
	},
	{
		area: areaDef{"Kota Bandung", "kota-bandung", "city", true, -6.9175, 107.6191}, priceMin: 1_000_000, priceMax: 2_700_000,
		spots: []spot{
			{&areaDef{"Institut Teknologi Bandung", "institut-teknologi-bandung", "campus", true, -6.8915, 107.6107}, -6.8915, 107.6107, "Coblong",
				[]string{"Dago", "Cisitu", "Sekeloa", "Tubagus Ismail"}, []string{"Jl. Ir. H. Juanda", "Jl. Cisitu Lama", "Jl. Tubagus Ismail", "Jl. Sekeloa"}},
			{nil, -6.9215, 107.6070, "Sumur Bandung", []string{"Braga", "Riau", "Cihapit"}, []string{"Jl. RE Martadinata", "Jl. Braga", "Jl. Cihapit"}},
		},
	},
	{
		area: areaDef{"Kota Yogyakarta", "kota-yogyakarta", "city", true, -7.7956, 110.3695}, priceMin: 650_000, priceMax: 1_800_000,
		spots: []spot{
			{&areaDef{"Universitas Gadjah Mada", "universitas-gadjah-mada", "campus", true, -7.7713, 110.3775}, -7.7713, 110.3775, "Depok, Sleman",
				[]string{"Pogung", "Sendowo", "Karangmalang", "Klebengan"}, []string{"Jl. Pogung Baru", "Jl. Kaliurang", "Jl. Colombo", "Jl. Sendowo"}},
			{nil, -7.7750, 110.4090, "Caturtunggal", []string{"Seturan", "Babarsari", "Gejayan"}, []string{"Jl. Seturan Raya", "Jl. Babarsari", "Jl. Affandi"}},
		},
	},
	{
		area: areaDef{"Kota Surabaya", "kota-surabaya", "city", true, -7.2575, 112.7521}, priceMin: 900_000, priceMax: 2_400_000,
		spots: []spot{
			{&areaDef{"Institut Teknologi Sepuluh Nopember", "institut-teknologi-sepuluh-nopember", "campus", true, -7.2819, 112.7949}, -7.2819, 112.7949, "Sukolilo",
				[]string{"Keputih", "Gebang", "Mulyosari"}, []string{"Jl. Keputih Tegal", "Jl. Gebang Lor", "Jl. Mulyosari"}},
			{nil, -7.2655, 112.7412, "Tegalsari", []string{"Darmo", "Kedungdoro", "Embong Malang"}, []string{"Jl. Raya Darmo", "Jl. Kedungdoro", "Jl. Embong Malang"}},
		},
	},
	{
		area: areaDef{"Kota Malang", "kota-malang", "city", false, -7.9666, 112.6326}, priceMin: 650_000, priceMax: 1_700_000,
		spots: []spot{
			{&areaDef{"Universitas Brawijaya", "universitas-brawijaya", "campus", true, -7.9525, 112.6144}, -7.9525, 112.6144, "Lowokwaru",
				[]string{"Dinoyo", "Ketawanggede", "Sumbersari", "Suhat"}, []string{"Jl. MT Haryono", "Jl. Kertoleksono", "Jl. Sumbersari", "Jl. Soekarno Hatta"}},
		},
	},
	{
		area: areaDef{"Kota Semarang", "kota-semarang", "city", false, -6.9667, 110.4167}, priceMin: 700_000, priceMax: 1_800_000,
		spots: []spot{
			{&areaDef{"Universitas Diponegoro", "universitas-diponegoro", "campus", false, -7.0505, 110.4381}, -7.0505, 110.4381, "Tembalang",
				[]string{"Tembalang", "Banjarsari", "Bulusan"}, []string{"Jl. Banjarsari", "Jl. Ngesrep Timur", "Jl. Bulusan Selatan"}},
		},
	},
}

var nameWords = []string{
	"Melati", "Mawar", "Anggrek", "Kenanga", "Cempaka", "Dahlia", "Flamboyan", "Seruni", "Teratai", "Bougenville",
	"Asri", "Harmoni", "Sejahtera", "Permata", "Cendana", "Pelangi", "Lavender", "Arjuna", "Srikandi", "Nusa Indah",
}

var namePrefixes = []string{"Kost", "Kost", "Kost", "Wisma", "Griya", "Rumah Kost", "Kos Eksklusif"}

var sharedFacilities = []string{
	"dapur bersama", "parkir motor", "CCTV 24 jam", "layanan laundry", "ruang tamu", "area jemur di rooftop", "mushola", "parkir mobil terbatas",
}

type roomTier struct {
	name       string
	multiplier float64
	facilities []string
	bedrooms   int
}

var roomTiers = []roomTier{
	{"Kamar Standar", 1.0, []string{"Kasur", "Lemari", "Meja Belajar", "WiFi", "Kipas Angin", "Kamar Mandi Luar"}, 1},
	{"Kamar Deluxe", 1.3, []string{"Kasur", "Lemari", "Meja Belajar", "WiFi", "AC", "Kamar Mandi Dalam"}, 2},
	{"Kamar VIP", 1.7, []string{"Kasur Queen", "Lemari", "Meja Kerja", "WiFi", "AC", "Kamar Mandi Dalam", "Water Heater", "TV", "Akses 24 Jam"}, 2},
}

var durationChoices = []string{`[1,3,6,12]`, `[1,6,12]`, `[3,6,12]`, `[1,3,6]`}

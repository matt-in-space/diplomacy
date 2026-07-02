package gamemap_test

var loadErrorCases = []struct {
	name string
	data string
	want string
}{
	{
		name: "empty nation id",
		data: `{
			"nations": [""],
			"provinces": [],
			"army_adjacency": {},
			"fleet_adjacency": {}
		}`,
		want: "nation id is required",
	},
	{
		name: "duplicate nation id",
		data: `{
			"nations": ["fra", "fra"],
			"provinces": [],
			"army_adjacency": {},
			"fleet_adjacency": {}
		}`,
		want: "duplicate nation",
	},
	{
		name: "unknown home nation",
		data: `{
			"nations": ["eng"],
			"provinces": [
				{ "id": "par", "name": "Paris", "type": "inland", "home_nation": "fra", "coasts": [] }
			],
			"army_adjacency": {},
			"fleet_adjacency": {}
		}`,
		want: "home nation \"fra\" not found",
	},
	{
		name: "duplicate province id",
		data: `{
			"provinces": [
				{ "id": "par", "name": "Paris", "type": "inland", "coasts": [] },
				{ "id": "par", "name": "Paris Again", "type": "inland", "coasts": [] }
			],
			"army_adjacency": {},
			"fleet_adjacency": {}
		}`,
		want: "duplicate province",
	},
	{
		name: "inland province with coast",
		data: `{
			"provinces": [{ "id": "par", "name": "Paris", "type": "inland", "coasts": ["par"] }],
			"army_adjacency": {},
			"fleet_adjacency": {}
		}`,
		want: "inland provinces cannot have coasts",
	},
	{
		name: "coastal province without coast",
		data: `{
			"provinces": [{ "id": "bre", "name": "Brest", "type": "coastal", "coasts": [] }],
			"army_adjacency": {},
			"fleet_adjacency": {}
		}`,
		want: "must have at least one coast",
	},
	{
		name: "duplicate coast id",
		data: `{
			"provinces": [
				{ "id": "bre", "name": "Brest", "type": "coastal", "coasts": ["bre"] },
				{ "id": "mao", "name": "Mid-Atlantic Ocean", "type": "water", "coasts": ["bre"] }
			],
			"army_adjacency": {},
			"fleet_adjacency": {}
		}`,
		want: "duplicate coast",
	},
	{
		name: "water province with army adjacency",
		data: `{
			"provinces": [
				{ "id": "par", "name": "Paris", "type": "inland", "coasts": [] },
				{ "id": "mao", "name": "Mid-Atlantic Ocean", "type": "water", "coasts": ["mao"] }
			],
			"army_adjacency": { "par": ["mao"], "mao": ["par"] },
			"fleet_adjacency": {}
		}`,
		want: "water provinces cannot have army adjacency",
	},
	{
		name: "unknown fleet coast",
		data: `{
			"provinces": [
				{ "id": "bre", "name": "Brest", "type": "coastal", "coasts": ["bre"] }
			],
			"army_adjacency": {},
			"fleet_adjacency": { "bre": ["missing"] }
		}`,
		want: "coast \"missing\" not found",
	},
	{
		name: "one way army adjacency",
		data: `{
			"provinces": [
				{ "id": "par", "name": "Paris", "type": "inland", "coasts": [] },
				{ "id": "bre", "name": "Brest", "type": "coastal", "coasts": ["bre"] }
			],
			"army_adjacency": { "par": ["bre"] },
			"fleet_adjacency": {}
		}`,
		want: "army adjacency",
	},
	{
		name: "one way fleet adjacency",
		data: `{
			"provinces": [
				{ "id": "bre", "name": "Brest", "type": "coastal", "coasts": ["bre"] },
				{ "id": "mao", "name": "Mid-Atlantic Ocean", "type": "water", "coasts": ["mao"] }
			],
			"army_adjacency": {},
			"fleet_adjacency": { "bre": ["mao"] }
		}`,
		want: "fleet adjacency",
	},
}

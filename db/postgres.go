package db

import (
	"encoding/json"
	"fitfuel/models"
	"log"
	"os"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://fitfuel:fitfuel123@localhost:5432/fitfuel"
	}

	database, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	DB = database
	log.Println("Database connected successfully")
}

func Migrate() {
	if err := DB.AutoMigrate(&models.Session{}, &models.Dish{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	backfillSessionFields()
	log.Println("Database migrations completed")
}

func Seed() {
	// Check if dishes already exist
	var count int64
	DB.Model(&models.Dish{}).Count(&count)
	if count > 0 {
		backfillDishAllergenTags()
		log.Println("Dishes already seeded, skipping...")
		return
	}

	dishes := getSeedDishes()
	for _, dish := range dishes {
		dish.AllergenTags = inferAllergenTags(dish.Ingredients)
		if err := DB.Create(&dish).Error; err != nil {
			log.Printf("Failed to seed dish %s: %v", dish.Name, err)
		}
	}
	backfillDishAllergenTags()
	log.Printf("Seeded %d dishes", len(dishes))
}

func backfillDishAllergenTags() {
	var dishes []models.Dish
	if err := DB.Find(&dishes).Error; err != nil {
		log.Printf("Failed to load dishes for allergen backfill: %v", err)
		return
	}

	updated := 0
	for _, dish := range dishes {
		if len(dish.AllergenTags) > 0 && string(dish.AllergenTags) != "[]" {
			continue
		}

		tags := inferAllergenTags(dish.Ingredients)
		if err := DB.Model(&models.Dish{}).
			Where("id = ?", dish.ID).
			Update("allergen_tags", tags).Error; err != nil {
			log.Printf("Failed to backfill allergen tags for dish %d: %v", dish.ID, err)
			continue
		}
		updated++
	}

	if updated > 0 {
		log.Printf("Backfilled allergen tags for %d dishes", updated)
	}
}

func backfillSessionFields() {
	if hasColumn("sessions", "heart_rate") {
		if err := DB.Exec("UPDATE sessions SET heart_rate = 0 WHERE heart_rate IS NULL").Error; err != nil {
			log.Printf("Failed to normalize heart_rate values: %v", err)
		}
	}

	if hasColumn("sessions", "restrictions") && hasColumn("sessions", "restrictions_text") {
		if err := DB.Exec(`
			UPDATE sessions
			SET restrictions_text = restrictions
			WHERE (restrictions_text IS NULL OR restrictions_text = '')
			  AND restrictions IS NOT NULL
			  AND restrictions <> ''
		`).Error; err != nil {
			log.Printf("Failed to backfill restrictions_text from restrictions: %v", err)
		}
	}

	if !hasColumn("sessions", "allergies") || !hasColumn("sessions", "restrictions_text") {
		return
	}

	type sessionLite struct {
		ID               string
		RestrictionsText string
		Allergies        datatypes.JSON
	}

	var sessions []sessionLite
	if err := DB.Raw("SELECT id, restrictions_text, allergies FROM sessions").Scan(&sessions).Error; err != nil {
		log.Printf("Failed to load sessions for allergy backfill: %v", err)
		return
	}

	updated := 0
	for _, session := range sessions {
		if len(session.Allergies) > 0 && string(session.Allergies) != "[]" {
			continue
		}

		allergies := parseCSVList(session.RestrictionsText)
		if len(allergies) == 0 {
			continue
		}

		raw, err := json.Marshal(allergies)
		if err != nil {
			continue
		}

		if err := DB.Model(&models.Session{}).
			Where("id = ?", session.ID).
			Update("allergies", raw).Error; err != nil {
			log.Printf("Failed to backfill allergies for session %s: %v", session.ID, err)
			continue
		}

		updated++
	}

	if updated > 0 {
		log.Printf("Backfilled allergies for %d sessions", updated)
	}
}

func parseCSVList(raw string) []string {
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		item := strings.ToLower(strings.TrimSpace(part))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		items = append(items, item)
	}

	return items
}

func hasColumn(tableName, columnName string) bool {
	var count int64
	err := DB.Raw(`
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = ?
		  AND column_name = ?
	`, tableName, columnName).Scan(&count).Error

	if err != nil {
		return false
	}

	return count > 0
}

func inferAllergenTags(ingredients datatypes.JSON) datatypes.JSON {
	var list []string
	if err := json.Unmarshal(ingredients, &list); err != nil {
		return datatypes.JSON([]byte("[]"))
	}

	joined := strings.ToLower(strings.Join(list, " "))
	tags := make([]string, 0)
	pushIfContains := func(tag string, keywords []string) {
		for _, keyword := range keywords {
			if strings.Contains(joined, keyword) {
				tags = append(tags, tag)
				return
			}
		}
	}

	pushIfContains("peanut", []string{"peanut", "groundnut"})
	pushIfContains("tree_nut", []string{"almond", "cashew", "walnut", "pistachio", "hazelnut"})
	pushIfContains("milk", []string{"milk", "paneer", "cheese", "yoghurt", "yogurt", "cream", "butter", "ghee", "mayo", "mayonnaise"})
	pushIfContains("egg", []string{" egg", "eggs", "omelette"})
	pushIfContains("soy", []string{"soy", "tofu"})
	pushIfContains("wheat", []string{"maida", "flour", "bread", "noodles", "pasta"})
	pushIfContains("sesame", []string{"sesame", "til"})
	pushIfContains("fish", []string{" fish", "fillet"})
	pushIfContains("shellfish", []string{"prawn", "shrimp", "crab", "lobster"})

	if len(tags) == 0 {
		return datatypes.JSON([]byte("[]"))
	}

	raw, err := json.Marshal(tags)
	if err != nil {
		return datatypes.JSON([]byte("[]"))
	}

	return raw
}

func getSeedDishes() []models.Dish {
	return []models.Dish{
		// VEG BREAKFAST
		{
			Name:        "Masala Oats",
			DietType:    "veg",
			MealType:    "Breakfast",
			Emoji:       "🌾",
			Calories:    190,
			Protein:     8,
			Carbs:       30,
			Fat:         4,
			PrepTime:    "10 min",
			Difficulty:  "Easy",
			Description: "Savoury rolled oats with vegetables and Indian spices — a quick high-fibre breakfast.",
			Ingredients: datatypes.JSON([]byte(`["1 cup rolled oats","1 tsp mustard seeds","1 small onion","1 carrot grated","½ tsp turmeric","Green chilli","Salt","Coriander"]`)),
			Steps:       datatypes.JSON([]byte(`["Heat oil, add mustard seeds until they pop.","Add onion and chilli, sauté 2 minutes.","Add carrot, turmeric, salt.","Add oats and 1.5 cups hot water, cook 4 min stirring.","Garnish with coriander."]`)),
		},
		{
			Name:        "Veggie Dosa",
			DietType:    "veg",
			MealType:    "Breakfast",
			Emoji:       "🥘",
			Calories:    220,
			Protein:     6,
			Carbs:       38,
			Fat:         5,
			PrepTime:    "30 min",
			Difficulty:  "Medium",
			Description: "Crispy South Indian crepe filled with spiced vegetables.",
			Ingredients: datatypes.JSON([]byte(`["1 cup dosa batter","1 onion sliced","1 carrot sliced","1 cup cabbage chopped","2 green chillies","Oil","Salt"]`)),
			Steps:       datatypes.JSON([]byte(`["Heat a non-stick pan.","Pour dosa batter in circular motion.","Add sautéed vegetables on one half.","Fold and remove when golden.","Serve with sambar or chutney."]`)),
		},
		// VEG LUNCH
		{
			Name:        "Dal Tadka",
			DietType:    "veg",
			MealType:    "Lunch",
			Emoji:       "🍛",
			Calories:    310,
			Protein:     16,
			Carbs:       42,
			Fat:         9,
			PrepTime:    "25 min",
			Difficulty:  "Easy",
			Description: "Creamy yellow lentils tempered with ghee, cumin, garlic and dried red chillies.",
			Ingredients: datatypes.JSON([]byte(`["1 cup yellow dal","2 tbsp ghee","1 tsp cumin","4 garlic cloves","2 dried red chillies","1 tsp turmeric","Salt","Coriander"]`)),
			Steps:       datatypes.JSON([]byte(`["Pressure cook dal with turmeric for 3 whistles.","Heat ghee, add cumin until it sputters.","Add garlic and red chillies, fry until golden.","Pour tadka over dal, mix well, garnish with coriander."]`)),
		},
		{
			Name:        "Chole Bhature",
			DietType:    "veg",
			MealType:    "Lunch",
			Emoji:       "🍲",
			Calories:    380,
			Protein:     12,
			Carbs:       52,
			Fat:         14,
			PrepTime:    "45 min",
			Difficulty:  "Hard",
			Description: "Spiced chickpeas with deep-fried puffed bread.",
			Ingredients: datatypes.JSON([]byte(`["2 cups cooked chickpeas","2 onions","3 tomatoes","Ginger-garlic paste","Flour","Yoghurt","Spices"]`)),
			Steps:       datatypes.JSON([]byte(`["Make dough with flour, yoghurt, salt. Let rest 2 hours.","Sauté onions until golden.","Add tomatoes, ginger-garlic, spices. Cook 15 min.","Add chickpeas, simmer.","Deep fry dough pieces until puffy."]`)),
		},
		// VEG DINNER
		{
			Name:        "Paneer Tikka Masala",
			DietType:    "veg",
			MealType:    "Dinner",
			Emoji:       "🍲",
			Calories:    350,
			Protein:     20,
			Carbs:       28,
			Fat:         18,
			PrepTime:    "40 min",
			Difficulty:  "Medium",
			Description: "Cottage cheese in creamy tomato and spice sauce.",
			Ingredients: datatypes.JSON([]byte(`["250g paneer cubed","2 cups tomato sauce","1 cup cream","Ginger-garlic paste","Onion","Spices"]`)),
			Steps:       datatypes.JSON([]byte(`["Sauté onion and ginger-garlic.","Add tomato sauce and spices.","Simmer 10 minutes.","Add paneer and cream.","Cook 5 more minutes. Serve hot."]`)),
		},
		{
			Name:        "Vegetable Biryani",
			DietType:    "veg",
			MealType:    "Dinner",
			Emoji:       "🍚",
			Calories:    340,
			Protein:     9,
			Carbs:       58,
			Fat:         8,
			PrepTime:    "60 min",
			Difficulty:  "Hard",
			Description: "Aromatic basmati rice layered with vegetables and spices.",
			Ingredients: datatypes.JSON([]byte(`["2 cups basmati rice","Mixed vegetables","Yoghurt","Whole spices","Saffron","Mint","Ghee"]`)),
			Steps:       datatypes.JSON([]byte(`["Fry onions golden, set aside.","Layer rice and vegetables.","Add yoghurt and spices.","Add fried onions and saffron.","Dum cook 25 minutes on low heat."]`)),
		},
		// VEG SNACKS
		{
			Name:        "Paneer Tikka",
			DietType:    "veg",
			MealType:    "Snack",
			Emoji:       "🧀",
			Calories:    280,
			Protein:     20,
			Carbs:       12,
			Fat:         17,
			PrepTime:    "30 min",
			Difficulty:  "Medium",
			Description: "Marinated cottage cheese cubes grilled with peppers and onions.",
			Ingredients: datatypes.JSON([]byte(`["250g paneer","1 cup yoghurt","2 tsp tandoori masala","1 tsp ginger-garlic paste","1 capsicum","1 onion","Chaat masala","Lemon"]`)),
			Steps:       datatypes.JSON([]byte(`["Mix yoghurt with spices and ginger-garlic paste.","Coat paneer and vegetables, marinate 20 min.","Skewer and grill at 200°C for 12-15 min.","Sprinkle chaat masala and lemon before serving."]`)),
		},
		{
			Name:        "Vegetable Samosa",
			DietType:    "veg",
			MealType:    "Snack",
			Emoji:       "🥟",
			Calories:    240,
			Protein:     4,
			Carbs:       32,
			Fat:         11,
			PrepTime:    "40 min",
			Difficulty:  "Medium",
			Description: "Crispy pastry pockets filled with spiced potatoes and peas.",
			Ingredients: datatypes.JSON([]byte(`["Maida flour","Potatoes","Peas","Onion","Green chillies","Spices","Oil"]`)),
			Steps:       datatypes.JSON([]byte(`["Make dough and rest 30 min.","Cook potatoes and mix with peas and spices.","Roll and cut dough into triangles.","Fill with vegetable mixture.","Deep fry until golden and crispy."]`)),
		},
		{
			Name:        "Guacamole Toast",
			DietType:    "veg",
			MealType:    "Snack",
			Emoji:       "🥑",
			Calories:    200,
			Protein:     6,
			Carbs:       24,
			Fat:         9,
			PrepTime:    "10 min",
			Difficulty:  "Easy",
			Description: "Creamy avocado spread on whole wheat toast with fresh toppings.",
			Ingredients: datatypes.JSON([]byte(`["1 avocado","2 slices whole wheat bread","Lemon juice","Tomato","Onion","Black pepper","Salt"]`)),
			Steps:       datatypes.JSON([]byte(`["Toast the bread until golden.","Mash avocado with lemon juice, salt, and pepper.","Spread on toast.","Top with tomato and onion.","Serve immediately."]`)),
		},
		// EGG BREAKFAST
		{
			Name:        "Egg Bhurji",
			DietType:    "egg",
			MealType:    "Breakfast",
			Emoji:       "🍳",
			Calories:    240,
			Protein:     18,
			Carbs:       8,
			Fat:         15,
			PrepTime:    "12 min",
			Difficulty:  "Easy",
			Description: "Spiced Indian scrambled eggs with onions, tomatoes and green chillies.",
			Ingredients: datatypes.JSON([]byte(`["3 eggs","1 onion","1 tomato","2 green chillies","½ tsp cumin","¼ tsp turmeric","Salt","Coriander"]`)),
			Steps:       datatypes.JSON([]byte(`["Heat oil, add cumin seeds.","Add onion and chillies, fry until golden.","Add tomato, cook until soft.","Pour beaten eggs, stir continuously until cooked.","Season and garnish."]`)),
		},
		{
			Name:        "Veggie Omelette",
			DietType:    "egg",
			MealType:    "Breakfast",
			Emoji:       "🥚",
			Calories:    210,
			Protein:     16,
			Carbs:       6,
			Fat:         13,
			PrepTime:    "15 min",
			Difficulty:  "Easy",
			Description: "Fluffy omelette filled with fresh vegetables and cheese.",
			Ingredients: datatypes.JSON([]byte(`["2 eggs","1 onion","1 capsicum","1 tomato","100g cheese","Butter","Salt","Pepper"]`)),
			Steps:       datatypes.JSON([]byte(`["Beat eggs with salt and pepper.","Heat butter in pan.","Pour eggs and spread evenly.","Add vegetables and cheese to one half.","Fold when set and serve hot."]`)),
		},
		// EGG LUNCH
		{
			Name:        "Egg Fried Rice",
			DietType:    "egg",
			MealType:    "Lunch",
			Emoji:       "🍚",
			Calories:    320,
			Protein:     12,
			Carbs:       42,
			Fat:         11,
			PrepTime:    "20 min",
			Difficulty:  "Easy",
			Description: "Fluffy rice with scrambled eggs and mixed vegetables.",
			Ingredients: datatypes.JSON([]byte(`["2 cups cooked rice","2 eggs","1 cup mixed vegetables","2 green onions","2 tbsp soy sauce","Oil","Garlic"]`)),
			Steps:       datatypes.JSON([]byte(`["Heat oil in wok, add garlic.","Scramble eggs and set aside.","Add vegetables, stir-fry 2 min.","Add rice and soy sauce.","Mix in eggs and green onions. Toss well."]`)),
		},
		{
			Name:        "Egg Curry",
			DietType:    "egg",
			MealType:    "Lunch",
			Emoji:       "🍛",
			Calories:    380,
			Protein:     14,
			Carbs:       24,
			Fat:         25,
			PrepTime:    "30 min",
			Difficulty:  "Medium",
			Description: "Boiled eggs in a rich, aromatic curry sauce.",
			Ingredients: datatypes.JSON([]byte(`["6 boiled eggs","2 onions","3 tomatoes","Coconut milk","Ginger-garlic paste","Curry leaves","Spices"]`)),
			Steps:       datatypes.JSON([]byte(`["Fry onions until golden.","Add ginger-garlic and spices.","Add tomatoes and cook 10 min.","Add coconut milk and bring to simmer.","Add halved boiled eggs and cook 5 min."]`)),
		},
		// EGG DINNER
		{
			Name:        "Egg Noodles",
			DietType:    "egg",
			MealType:    "Dinner",
			Emoji:       "🍜",
			Calories:    350,
			Protein:     14,
			Carbs:       44,
			Fat:         12,
			PrepTime:    "20 min",
			Difficulty:  "Easy",
			Description: "Stir-fried egg noodles with vegetables and light sauce.",
			Ingredients: datatypes.JSON([]byte(`["200g egg noodles","2 eggs","Mixed vegetables","Green onions","Soy sauce","Oil","Garlic chilli sauce"]`)),
			Steps:       datatypes.JSON([]byte(`["Cook noodles, drain and set aside.","Heat oil, scramble eggs.","Add vegetables, stir-fry.","Add noodles and sauces.","Toss everything together. Serve hot."]`)),
		},
		{
			Name:        "Shakshuka",
			DietType:    "egg",
			MealType:    "Dinner",
			Emoji:       "🍲",
			Calories:    340,
			Protein:     12,
			Carbs:       22,
			Fat:         22,
			PrepTime:    "25 min",
			Difficulty:  "Medium",
			Description: "Eggs poached in spiced tomato sauce.",
			Ingredients: datatypes.JSON([]byte(`["4 eggs","3 tomatoes","1 onion","2 red chillies","Garlic","Paprika","Olive oil","Parsley"]`)),
			Steps:       datatypes.JSON([]byte(`["Heat oil, fry onion and garlic.","Add tomatoes and paprika, cook 5 min.","Make 4 wells in sauce.","Crack eggs into wells.","Cover and cook until eggs set 10 min."]`)),
		},
		// EGG SNACKS
		{
			Name:        "Egg Spring Rolls",
			DietType:    "egg",
			MealType:    "Snack",
			Emoji:       "🥟",
			Calories:    260,
			Protein:     10,
			Carbs:       28,
			Fat:         12,
			PrepTime:    "30 min",
			Difficulty:  "Medium",
			Description: "Crispy rolls filled with scrambled eggs and vegetables.",
			Ingredients: datatypes.JSON([]byte(`["2 eggs","Spring roll wrappers","Cabbage","Carrots","Green onions","Soy sauce","Oil"]`)),
			Steps:       datatypes.JSON([]byte(`["Scramble eggs and cool.","Mix with shredded vegetables.","Place filling on wrapper.","Roll tightly and seal.","Deep fry until golden."]`)),
		},
		{
			Name:        "Egg Salad Sandwich",
			DietType:    "egg",
			MealType:    "Snack",
			Emoji:       "🥪",
			Calories:    280,
			Protein:     12,
			Carbs:       32,
			Fat:         12,
			PrepTime:    "15 min",
			Difficulty:  "Easy",
			Description: "Creamy egg salad on whole wheat bread.",
			Ingredients: datatypes.JSON([]byte(`["3 boiled eggs","2 slices whole wheat bread","Mayonnaise","Lettuce","Tomato","Salt","Pepper"]`)),
			Steps:       datatypes.JSON([]byte(`["Chop boiled eggs.","Mix with mayonnaise, salt, and pepper.","Toast bread.","Layer with lettuce, tomato, and egg mixture.","Serve fresh."]`)),
		},
		{
			Name:        "Deviled Eggs",
			DietType:    "egg",
			MealType:    "Snack",
			Emoji:       "🥚",
			Calories:    190,
			Protein:     15,
			Carbs:       2,
			Fat:         14,
			PrepTime:    "20 min",
			Difficulty:  "Easy",
			Description: "Boiled eggs topped with spiced creamy filling.",
			Ingredients: datatypes.JSON([]byte(`["6 boiled eggs","Mayonnaise","Mustard","Paprika","Chaat masala","Lemon juice","Salt"]`)),
			Steps:       datatypes.JSON([]byte(`["Cut boiled eggs in half.","Scoop out yolks.","Mix yolks with mayo, mustard, and spices.","Fill egg white halves.","Top with paprika and serve chilled."]`)),
		},
		// NON-VEG BREAKFAST
		{
			Name:        "Chicken Omelette",
			DietType:    "nonveg",
			MealType:    "Breakfast",
			Emoji:       "🍳",
			Calories:    280,
			Protein:     28,
			Carbs:       6,
			Fat:         16,
			PrepTime:    "20 min",
			Difficulty:  "Medium",
			Description: "Fluffy omelette stuffed with tender chicken pieces and vegetables.",
			Ingredients: datatypes.JSON([]byte(`["2 eggs","100g cooked chicken","1 onion","1 capsicum","100g cheese","Butter","Salt","Pepper"]`)),
			Steps:       datatypes.JSON([]byte(`["Dice cooked chicken.","Beat eggs with salt and pepper.","Heat butter and pour eggs.","Add chicken and vegetables.","Fold and serve immediately."]`)),
		},
		{
			Name:        "Bacon & Egg Breakfast",
			DietType:    "nonveg",
			MealType:    "Breakfast",
			Emoji:       "🥓",
			Calories:    320,
			Protein:     24,
			Carbs:       12,
			Fat:         20,
			PrepTime:    "15 min",
			Difficulty:  "Easy",
			Description: "Crispy bacon strips with fried eggs and whole wheat toast.",
			Ingredients: datatypes.JSON([]byte(`["3 bacon strips","2 eggs","2 slices whole wheat toast","Butter","Salt","Pepper"]`)),
			Steps:       datatypes.JSON([]byte(`["Fry bacon until crispy.","Remove and set aside.","Fry eggs in bacon fat.","Toast bread.","Assemble and serve hot."]`)),
		},
		// NON-VEG LUNCH
		{
			Name:        "Butter Chicken",
			DietType:    "nonveg",
			MealType:    "Lunch",
			Emoji:       "🍗",
			Calories:    420,
			Protein:     32,
			Carbs:       20,
			Fat:         24,
			PrepTime:    "45 min",
			Difficulty:  "Medium",
			Description: "Tender chicken in a creamy tomato and butter sauce.",
			Ingredients: datatypes.JSON([]byte(`["500g chicken","3 tomatoes","1 cup cream","4 tbsp butter","Ginger-garlic paste","Spices","Lemon"]`)),
			Steps:       datatypes.JSON([]byte(`["Marinate and grill chicken.","Puree tomatoes.","Sauté in butter with ginger-garlic.","Add tomato puree and spices.","Add cream and chicken. Cook 10 min."]`)),
		},
		{
			Name:        "Fish Biryani",
			DietType:    "nonveg",
			MealType:    "Lunch",
			Emoji:       "🍚",
			Calories:    380,
			Protein:     28,
			Carbs:       48,
			Fat:         12,
			PrepTime:    "50 min",
			Difficulty:  "Hard",
			Description: "Aromatic basmati rice layered with marinated fish.",
			Ingredients: datatypes.JSON([]byte(`["500g fish","2 cups basmati rice","Yoghurt","Whole spices","Saffron","Mint","Fried onions","Ghee"]`)),
			Steps:       datatypes.JSON([]byte(`["Marinate fish in yoghurt and spices.","Fry onions golden.","Parboil rice.","Layer and add saffron.","Dum cook 25 minutes."]`)),
		},
		// NON-VEG DINNER
		{
			Name:        "Grilled Fish",
			DietType:    "nonveg",
			MealType:    "Dinner",
			Emoji:       "🐟",
			Calories:    290,
			Protein:     34,
			Carbs:       4,
			Fat:         14,
			PrepTime:    "20 min",
			Difficulty:  "Easy",
			Description: "Lemon herb grilled fish fillet — low calorie, high protein.",
			Ingredients: datatypes.JSON([]byte(`["2 fish fillets","2 tbsp olive oil","1 lemon","3 garlic cloves","Fresh parsley","½ tsp paprika","Salt","Black pepper"]`)),
			Steps:       datatypes.JSON([]byte(`["Pat fish dry.","Mix oil, lemon zest, garlic, paprika, parsley.","Coat fish and marinate 15 min.","Grill 3-4 min per side.","Squeeze lemon over, serve."]`)),
		},
		{
			Name:        "Tandoori Chicken",
			DietType:    "nonveg",
			MealType:    "Dinner",
			Emoji:       "🍗",
			Calories:    340,
			Protein:     38,
			Carbs:       8,
			Fat:         16,
			PrepTime:    "40 min",
			Difficulty:  "Medium",
			Description: "Spiced chicken marinated and cooked in tandoor.",
			Ingredients: datatypes.JSON([]byte(`["600g chicken","1 cup yoghurt","3 tbsp tandoori masala","Ginger-garlic paste","Lemon","Oil","Green chillies"]`)),
			Steps:       datatypes.JSON([]byte(`["Mix yoghurt with tandoori masala and ginger-garlic.","Marinate chicken 30 min.","Thread on skewers.","Grill at 200°C for 20-25 min.","Brush with oil, serve hot."]`)),
		},
		// NON-VEG SNACKS
		{
			Name:        "Chicken Samosa",
			DietType:    "nonveg",
			MealType:    "Snack",
			Emoji:       "🥟",
			Calories:    280,
			Protein:     14,
			Carbs:       30,
			Fat:         12,
			PrepTime:    "45 min",
			Difficulty:  "Medium",
			Description: "Crispy pastry pockets filled with spiced minced chicken.",
			Ingredients: datatypes.JSON([]byte(`["300g minced chicken","Maida flour","Onion","Green chillies","Ginger-garlic paste","Spices","Oil"]`)),
			Steps:       datatypes.JSON([]byte(`["Cook minced chicken with spices.","Make dough and rest.","Roll and cut triangles.","Fill with chicken mixture.","Deep fry until golden."]`)),
		},
		{
			Name:        "Fish Cakes",
			DietType:    "nonveg",
			MealType:    "Snack",
			Emoji:       "🍤",
			Calories:    240,
			Protein:     18,
			Carbs:       20,
			Fat:         11,
			PrepTime:    "30 min",
			Difficulty:  "Medium",
			Description: "Pan-fried cakes made with flaked fish and potatoes.",
			Ingredients: datatypes.JSON([]byte(`["300g fish","2 potatoes","1 onion","Green chillies","Breadcrumbs","Egg","Oil","Lemon"]`)),
			Steps:       datatypes.JSON([]byte(`["Boil and mash potatoes.","Mix with flaked fish and onion.","Form patties and coat with breadcrumbs.","Pan-fry until golden.","Serve with lemon."]`)),
		},
		{
			Name:        "Chicken Kebab",
			DietType:    "nonveg",
			MealType:    "Snack",
			Emoji:       "🍢",
			Calories:    220,
			Protein:     25,
			Carbs:       6,
			Fat:         11,
			PrepTime:    "25 min",
			Difficulty:  "Medium",
			Description: "Seasoned ground chicken formed into skewers and grilled.",
			Ingredients: datatypes.JSON([]byte(`["400g minced chicken","Ginger-garlic paste","Green chillies","Onion","Yoghurt","Spices","Bamboo skewers"]`)),
			Steps:       datatypes.JSON([]byte(`["Mix chicken with paste and spices.","Form onto bamboo skewers.","Grill 4-5 min per side.","Brush with oil.","Serve hot with chutney."]`)),
		},
	}
}

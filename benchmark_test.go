package rules

import (
	"context"
	"fmt"
	"testing"
)

type benchUser struct {
	ID    int
	Email string
	Age   int
}

type benchProduct struct {
	ID    int
	Name  string
	Price float64
}

// BenchmarkConditionTypeCheck compares different type checking approaches
func BenchmarkConditionTypeCheck(b *testing.B) {
	user := benchUser{ID: 1, Email: "test@example.com", Age: 25}
	ctx := WithRegistry(context.Background(), NewDataRegistry(user))

	b.Run("IsA_with_reflection", func(b *testing.B) {
		cond := IsA[benchUser]("isUser")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cond.IsValid(ctx)
		}
	})

	b.Run("FastIsA_generic", func(b *testing.B) {
		cond := FastIsA[benchUser]("isUser")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cond.IsValid(ctx)
		}
	})

	b.Run("FastTypeSwitch", func(b *testing.B) {
		cond := FastTypeSwitch("isUser", func(data any) bool {
			switch data.(type) {
			case benchUser, *benchUser:
				return true
			default:
				return false
			}
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cond.IsValid(ctx)
		}
	})

	b.Run("NewCondition_with_type_assertion", func(b *testing.B) {
		cond := NewCondition("isUser", func(ctx context.Context) bool {
			_, ok := GetAs[benchUser](ctx)
			return ok
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cond.IsValid(ctx)
		}
	})

	b.Run("NewCondition_with_direct_cast", func(b *testing.B) {
		cond := NewCondition("isUser", func(ctx context.Context) bool {
			data, ok := Get(ctx)
			if !ok {
				return false
			}
			_, ok = data.(benchUser)
			return ok
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cond.IsValid(ctx)
		}
	})
}

// BenchmarkFullTree compares full tree evaluation with different approaches
func BenchmarkFullTree(b *testing.B) {
	users := make([]benchUser, 1000)
	for i := range users {
		users[i] = benchUser{
			ID:    i,
			Email: fmt.Sprintf("user%d@example.com", i),
			Age:   18 + (i % 50),
		}
	}

	b.Run("IsA_reflection", func(b *testing.B) {
		tree := Node(
			IsA[benchUser]("isUser"),
			Rules(
				NewTypedRule[benchUser]("checkAge", func(ctx context.Context, u benchUser) error {
					if u.Age < 18 {
						return fmt.Errorf("too young")
					}
					return nil
				}),
			),
		)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			user := users[i%len(users)]
			_ = ValidateWithData(context.Background(), tree, ProcessingHooks{}, "test", user)
		}
	})

	b.Run("FastTypeSwitch", func(b *testing.B) {
		tree := Node(
			FastTypeSwitch("isUser", func(data any) bool {
				switch data.(type) {
				case benchUser, *benchUser:
					return true
				default:
					return false
				}
			}),
			Rules(
				NewTypedRule[benchUser]("checkAge", func(ctx context.Context, u benchUser) error {
					if u.Age < 18 {
						return fmt.Errorf("too young")
					}
					return nil
				}),
			),
		)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			user := users[i%len(users)]
			_ = ValidateWithData(context.Background(), tree, ProcessingHooks{}, "test", user)
		}
	})

	b.Run("NewCondition_direct", func(b *testing.B) {
		tree := Node(
			NewCondition("isUser", func(ctx context.Context) bool {
				data, ok := Get(ctx)
				if !ok {
					return false
				}
				_, ok = data.(benchUser)
				return ok
			}),
			Rules(
				NewTypedRule[benchUser]("checkAge", func(ctx context.Context, u benchUser) error {
					if u.Age < 18 {
						return fmt.Errorf("too young")
					}
					return nil
				}),
			),
		)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			user := users[i%len(users)]
			_ = ValidateWithData(context.Background(), tree, ProcessingHooks{}, "test", user)
		}
	})
}

// BenchmarkTypeSwitchVsReflection compares type switch vs reflection for multiple types
func BenchmarkTypeSwitchVsReflection(b *testing.B) {
	users := make([]benchUser, 100)
	products := make([]benchProduct, 100)
	mixed := make([]any, 200)

	for i := 0; i < 100; i++ {
		users[i] = benchUser{ID: i, Age: 25}
		products[i] = benchProduct{ID: i, Price: 10.99}
		mixed[i*2] = users[i]
		mixed[i*2+1] = products[i]
	}

	b.Run("IsA_with_reflection", func(b *testing.B) {
		userCond := IsA[benchUser]("isUser")
		productCond := IsA[benchProduct]("isProduct")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data := mixed[i%len(mixed)]
			ctx := WithRegistry(context.Background(), NewDataRegistry(data))
			_ = userCond.IsValid(ctx)
			_ = productCond.IsValid(ctx)
		}
	})

	b.Run("FastTypeSwitch", func(b *testing.B) {
		cond := FastTypeSwitch("typeCheck", func(data any) bool {
			switch data.(type) {
			case benchUser, *benchUser, benchProduct, *benchProduct:
				return true
			default:
				return false
			}
		})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data := mixed[i%len(mixed)]
			ctx := WithRegistry(context.Background(), NewDataRegistry(data))
			_ = cond.IsValid(ctx)
		}
	})

	b.Run("Direct_type_assertion", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data := mixed[i%len(mixed)]
			switch data.(type) {
			case benchUser, *benchUser, benchProduct, *benchProduct:
				// matched
			default:
				// not matched
			}
		}
	})
}

// BenchmarkMergedTrees compares evaluating merged trees with type switching
func BenchmarkMergedTrees(b *testing.B) {
	users := make([]benchUser, 1000)
	products := make([]benchProduct, 1000)
	for i := 0; i < 1000; i++ {
		users[i] = benchUser{ID: i, Email: "test@example.com", Age: 25}
		products[i] = benchProduct{ID: i, Name: "Product", Price: 10.99}
	}

	userTree := Node(
		IsA[benchUser]("isUser"),
		Rules(
			NewTypedRule[benchUser]("checkEmail", func(ctx context.Context, u benchUser) error {
				if u.Email == "" {
					return fmt.Errorf("email required")
				}
				return nil
			}),
		),
	)

	productTree := Node(
		IsA[benchProduct]("isProduct"),
		Rules(
			NewTypedRule[benchProduct]("checkPrice", func(ctx context.Context, p benchProduct) error {
				if p.Price <= 0 {
					return fmt.Errorf("price must be positive")
				}
				return nil
			}),
		),
	)

	mergedTree := Root(userTree, productTree)

	b.Run("mixed_users_and_products", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if i%2 == 0 {
				_ = ValidateWithData(context.Background(), mergedTree, ProcessingHooks{}, "test", users[i%len(users)])
			} else {
				_ = ValidateWithData(context.Background(), mergedTree, ProcessingHooks{}, "test", products[i%len(products)])
			}
		}
	})

	b.Run("users_only", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ValidateWithData(context.Background(), mergedTree, ProcessingHooks{}, "test", users[i%len(users)])
		}
	})

	b.Run("products_only", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ValidateWithData(context.Background(), mergedTree, ProcessingHooks{}, "test", products[i%len(products)])
		}
	})
}

// BenchmarkDataAccess compares different ways to access data from context
func BenchmarkDataAccess(b *testing.B) {
	user := benchUser{ID: 1, Email: "test@example.com", Age: 25}
	ctx := WithRegistry(context.Background(), NewDataRegistry(user))

	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = Get(ctx)
		}
	})

	b.Run("GetAs", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = GetAs[benchUser](ctx)
		}
	})

	b.Run("Get_with_type_assertion", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data, _ := Get(ctx)
			_ = data.(benchUser)
		}
	})
}

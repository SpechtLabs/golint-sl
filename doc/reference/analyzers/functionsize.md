---
title: functionsize
permalink: /reference/analyzers/functionsize
createTime: 2025/01/16 10:00:00
---

Enforces function length limits with refactoring suggestions.

## Category

Architecture

## What It Checks

This analyzer detects functions that are too long and suggests refactoring approaches.

## Why It Matters

Long functions:

- Are hard to understand
- Are hard to test
- Often do too many things
- Hide bugs in complexity

## Examples

### Bad: Long Function

```go
func ProcessOrder(ctx context.Context, order *Order) error {
    // Validate order (20 lines)
    if order == nil {
        return errors.New("order is nil")
    }
    if order.UserID == "" {
        return errors.New("user ID required")
    }
    // ... 18 more validation lines ...

    // Fetch user (15 lines)
    user, err := db.GetUser(ctx, order.UserID)
    if err != nil {
        return err
    }
    // ... more user logic ...

    // Check inventory (25 lines)
    // ...

    // Process payment (30 lines)
    // ...

    // Update order (20 lines)
    // ...

    // Send notifications (15 lines)
    // ...

    return nil
}
```

### Good: Extracted Functions

```go
func ProcessOrder(ctx context.Context, order *Order) error {
    if err := validateOrder(order); err != nil {
        return fmt.Errorf("validate: %w", err)
    }

    user, err := fetchUser(ctx, order.UserID)
    if err != nil {
        return fmt.Errorf("fetch user: %w", err)
    }

    if err := checkInventory(ctx, order.Items); err != nil {
        return fmt.Errorf("inventory: %w", err)
    }

    if err := processPayment(ctx, user, order); err != nil {
        return fmt.Errorf("payment: %w", err)
    }

    if err := updateOrder(ctx, order); err != nil {
        return fmt.Errorf("update: %w", err)
    }

    if err := sendNotifications(ctx, user, order); err != nil {
        // Non-critical, log but don't fail
        log.Error("notifications failed", err)
    }

    return nil
}

func validateOrder(order *Order) error {
    if order == nil {
        return errors.New("order is nil")
    }
    if order.UserID == "" {
        return errors.New("user ID required")
    }
    // ... focused validation logic ...
    return nil
}

// ... other extracted functions ...
```

## Refactoring Strategies

1. **Extract by Responsibility**: Group related logic into functions
2. **Extract by Abstraction Level**: High-level orchestration vs. low-level details
3. **Extract by Testability**: Make complex logic independently testable

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  functionsize: true  # enabled by default
```

## When to Disable

- Generated code
- Data initialization functions
- Test functions with many cases

```yaml
analyzers:
  functionsize: false
```

## Related Analyzers

- [nestingdepth](/reference/analyzers/nestingdepth) - Nesting limits
- [closurecomplexity](/reference/analyzers/closurecomplexity) - Closure complexity

# 06 — Finance Module

## Overview
Complete fee management: fee categories, fee templates, invoice generation, payment tracking. Revenue-critical for schools.

---

## Backend Changes

### 1. Database Tables (New Migration)
**File:** `backend/db/migrations/XXXXXX_create_finance.up.sql`

```sql
-- Fee categories (e.g., Tuition, Transport, Uniform, Meals)
CREATE TABLE fee_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    is_recurring BOOLEAN NOT NULL DEFAULT TRUE,
    frequency TEXT, -- 'MONTHLY', 'TERMLY', 'ANNUALLY', 'ONE_TIME'
    default_amount NUMERIC,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, name)
);

-- Fee templates (applied to classes/grades)
CREATE TABLE fee_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id),
    applies_to_grade_levels UUID[] NOT NULL DEFAULT '{}',
    applies_to_streams UUID[] NOT NULL DEFAULT '{}',
    items JSONB NOT NULL DEFAULT '[]', -- [{category_id, amount, is_optional, description}]
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, name, academic_year_id)
);

-- Invoices (generated per student)
CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    fee_template_id UUID REFERENCES fee_templates(id) ON DELETE SET NULL,
    invoice_number TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'DRAFT', -- 'DRAFT', 'SENT', 'PARTIAL', 'PAID', 'OVERDUE', 'CANCELLED'
    issue_date DATE NOT NULL DEFAULT CURRENT_DATE,
    due_date DATE NOT NULL,
    subtotal NUMERIC NOT NULL DEFAULT 0,
    discount NUMERIC NOT NULL DEFAULT 0,
    tax NUMERIC NOT NULL DEFAULT 0,
    total NUMERIC NOT NULL DEFAULT 0,
    paid_amount NUMERIC NOT NULL DEFAULT 0,
    balance NUMERIC NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, invoice_number)
);

-- Invoice line items
CREATE TABLE invoice_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    fee_category_id UUID NOT NULL REFERENCES fee_categories(id),
    description TEXT,
    quantity INT NOT NULL DEFAULT 1,
    unit_price NUMERIC NOT NULL,
    discount NUMERIC NOT NULL DEFAULT 0,
    tax_rate NUMERIC NOT NULL DEFAULT 0,
    total NUMERIC NOT NULL,
    sort_order INT NOT NULL DEFAULT 0
);

-- Payments
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    amount NUMERIC NOT NULL,
    payment_date DATE NOT NULL DEFAULT CURRENT_DATE,
    payment_method TEXT NOT NULL, -- 'CASH', 'CARD', 'BANK_TRANSFER', 'MOBILE_MONEY', 'OTHER'
    reference TEXT,
    received_by UUID REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'COMPLETED', -- 'COMPLETED', 'PENDING', 'FAILED', 'REFUNDED'
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_fee_categories_school ON fee_categories(school_id);
CREATE INDEX idx_fee_templates_school_year ON fee_templates(school_id, academic_year_id);
CREATE INDEX idx_invoices_student ON invoices(student_id);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_due_date ON invoices(due_date);
CREATE INDEX idx_invoice_items_invoice ON invoice_items(invoice_id);
CREATE INDEX idx_payments_invoice ON payments(invoice_id);
CREATE INDEX idx_payments_student ON payments(student_id);
CREATE INDEX idx_payments_date ON payments(payment_date);
```

### 2. SQLC Queries
**Files:**
- `backend/db/queries/fee_categories.sql`
- `backend/db/queries/fee_templates.sql`
- `backend/db/queries/invoices.sql`
- `backend/db/queries/payments.sql`

### 3. Services
- `FeeCategoriesService` — CRUD
- `FeeTemplatesService` — CRUD + apply to students
- `InvoicesService` — Generate, list, update status, bulk operations
- `PaymentsService` — Record payment, allocate to invoices, receipts

### 4. Handlers
- `FeeCategoriesHandler`
- `FeeTemplatesHandler`
- `InvoicesHandler`
- `PaymentsHandler`

### 5. Router Endpoints
```
# Fee Categories
GET    /api/finance/fee-categories
POST   /api/finance/fee-categories
PATCH  /api/finance/fee-categories/:id
DELETE /api/finance/fee-categories/:id

# Fee Templates
GET    /api/finance/fee-templates
POST   /api/finance/fee-templates
GET    /api/finance/fee-templates/:id
PATCH  /api/finance/fee-templates/:id
DELETE /api/finance/fee-templates/:id
POST   /api/finance/fee-templates/:id/apply   # Generate invoices for matching students

# Invoices
GET    /api/finance/invoices
POST   /api/finance/invoices
GET    /api/finance/invoices/:id
PATCH  /api/finance/invoices/:id
DELETE /api/finance/invoices/:id
POST   /api/finance/invoices/bulk-generate
GET    /api/finance/invoices/:id/pdf

# Payments
GET    /api/finance/payments
POST   /api/finance/payments
GET    /api/finance/payments/:id
PATCH  /api/finance/payments/:id
GET    /api/finance/students/:id/statement
```

---

## Frontend Changes

### 1. Feature Module
```
src/features/finance/
├── components/
│   ├── fee-categories-table.tsx
│   ├── fee-category-form.tsx
│   ├── fee-templates-table.tsx
│   ├── fee-template-form.tsx
│   ├── invoices-table.tsx
│   ├── invoice-form.tsx
│   ├── invoice-detail.tsx
│   ├── payments-table.tsx
│   ├── payment-form.tsx
│   └── student-statement.tsx
├── hooks/
│   ├── use-fee-categories.ts
│   ├── use-fee-templates.ts
│   ├── use-invoices.ts
│   └── use-payments.ts
├── services/
│   └── api.ts
├── types/
│   └── finance.ts
└── index.ts
```

### 2. Pages
```
src/app/(dashboard)/finance/
├── fee-categories/page.tsx
├── fee-templates/page.tsx
├── invoices/page.tsx
├── invoices/[id]/page.tsx
├── payments/page.tsx
└── students/[id]/statement/page.tsx
```

### 3. Navigation
Already in nav:
```tsx
{
    title: "Finance",
    url: "#",
    icon: <DollarSignIcon />,
    items: [
        { title: "Fee Categories", url: "/finance/fee-categories" },
        { title: "Fee Templates", url: "/finance/fee-templates" },
        { title: "Invoices", url: "/finance/invoices" },
    ],
}
```

---

## Acceptance Criteria
- [ ] Fee categories: create, edit, toggle active, set default amount
- [ ] Fee templates: define per academic year, filter by grade/stream
- [ ] Apply template → bulk generate invoices for matching students
- [ ] Invoice management: view, edit, send, cancel, PDF export
- [ ] Payment recording: multiple methods, partial payments, receipts
- [ ] Student statement: all invoices, payments, balance
- [ ] Dashboard: total outstanding, overdue, collected this month
- [ ] RLS: tenant isolation

---

## Dependencies
- Requires: Students, Classes, Academic Years, Grade Levels, Streams
- Integrates with: Student enrollment (for auto-invoice generation)

---

## Estimated Effort
- Backend: ~18 hours (4 tables, invoice generation logic)
- Frontend: ~15 hours (4 pages, invoice detail, statements)
- **Total: ~33 hours**
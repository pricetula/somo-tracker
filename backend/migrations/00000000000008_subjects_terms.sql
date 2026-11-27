-- +goose Up

-- Subjects: default subject catalog per education system per level.
CREATE TABLE subjects (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    education_system_id UUID NOT NULL REFERENCES education_systems(id) ON DELETE CASCADE,
    education_level_id UUID NOT NULL REFERENCES education_levels(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    is_compulsory_default BOOL NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (education_system_id, education_level_id, code),
    FOREIGN KEY (education_level_id, education_system_id)
        REFERENCES education_levels(id, education_system_id)
);

CREATE INDEX idx_subjects_system ON subjects (education_system_id);
CREATE INDEX idx_subjects_level ON subjects (education_level_id);

-- Topics: topics within a subject.
CREATE TABLE topics (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    sort_order INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (subject_id, name)
);

CREATE INDEX idx_topics_subject ON topics (subject_id);

-- Sub-topics: sub-topics within a topic.
CREATE TABLE sub_topics (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    topic_id UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    sort_order INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (topic_id, name)
);

CREATE INDEX idx_sub_topics_topic ON sub_topics (topic_id);

-- +goose Down

DROP TABLE IF EXISTS sub_topics;

DROP TABLE IF EXISTS topics;

DROP TABLE IF EXISTS subjects;

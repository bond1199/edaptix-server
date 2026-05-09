-- edaptix-server 初始化数据库迁移
-- 版本: 000001
-- 日期: 2026-05-09

-- 用户表
CREATE TABLE users (
    id              BIGSERIAL PRIMARY KEY,
    phone           VARCHAR(20) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'student',
    status          SMALLINT NOT NULL DEFAULT 1,
    initialized     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);
CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_role ON users(role);

-- 学生档案表
CREATE TABLE student_profiles (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT UNIQUE NOT NULL REFERENCES users(id),
    real_name       VARCHAR(50),
    grade           SMALLINT NOT NULL,
    grade_stage     VARCHAR(10) NOT NULL,
    school_name     VARCHAR(100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 家长账号表
CREATE TABLE parent_accounts (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT UNIQUE NOT NULL REFERENCES users(id),
    real_name       VARCHAR(50),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 家长-学生绑定表
CREATE TABLE parent_student_bindings (
    id              BIGSERIAL PRIMARY KEY,
    parent_id       BIGINT NOT NULL REFERENCES parent_accounts(id),
    student_id      BIGINT NOT NULL REFERENCES users(id),
    bind_code       VARCHAR(10) UNIQUE,
    bind_qrcode     VARCHAR(255),
    status          SMALLINT NOT NULL DEFAULT 1,
    bound_at        TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(parent_id, student_id)
);
CREATE INDEX idx_psb_student ON parent_student_bindings(student_id);
CREATE INDEX idx_psb_bind_code ON parent_student_bindings(bind_code);

-- 学科知识树表
CREATE TABLE knowledge_trees (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    subject         VARCHAR(30) NOT NULL,
    grade           SMALLINT NOT NULL,
    textbook_edition VARCHAR(50),
    status          SMALLINT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_kt_user ON knowledge_trees(user_id);

-- 知识节点表
CREATE TABLE knowledge_nodes (
    id              BIGSERIAL PRIMARY KEY,
    tree_id         BIGINT NOT NULL REFERENCES knowledge_trees(id) ON DELETE CASCADE,
    parent_id       BIGINT REFERENCES knowledge_nodes(id) ON DELETE CASCADE,
    level           SMALLINT NOT NULL,
    name            VARCHAR(200) NOT NULL,
    sort_order      INT NOT NULL DEFAULT 0,
    mastery_rate    DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    question_count  INT NOT NULL DEFAULT 0,
    correct_count   INT NOT NULL DEFAULT 0,
    error_count     INT NOT NULL DEFAULT 0,
    last_practiced  TIMESTAMPTZ,
    is_locked       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_kn_tree ON knowledge_nodes(tree_id);
CREATE INDEX idx_kn_parent ON knowledge_nodes(parent_id);
CREATE INDEX idx_kn_level ON knowledge_nodes(level);
CREATE INDEX idx_kn_mastery ON knowledge_nodes(mastery_rate);

-- 上传批次表
CREATE TABLE learning_uploads (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    upload_type     VARCHAR(20) NOT NULL,
    source          VARCHAR(20) NOT NULL,
    subject         VARCHAR(30) NOT NULL DEFAULT '',
    status          SMALLINT NOT NULL DEFAULT 1,
    page_count      INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_lu_user ON learning_uploads(user_id);
CREATE INDEX idx_lu_status ON learning_uploads(status);

-- 上传素材明细表
CREATE TABLE upload_items (
    id              BIGSERIAL PRIMARY KEY,
    upload_id       BIGINT NOT NULL REFERENCES learning_uploads(id) ON DELETE CASCADE,
    image_url       VARCHAR(500) NOT NULL,
    page_index      INT NOT NULL DEFAULT 0,
    is_valid        BOOLEAN NOT NULL DEFAULT TRUE,
    invalid_reason  VARCHAR(50),
    ocr_result      JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ui_upload ON upload_items(upload_id);

-- 错题表
CREATE TABLE error_questions (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    subject         VARCHAR(30) NOT NULL,
    knowledge_node_id BIGINT REFERENCES knowledge_nodes(id),
    question_type   VARCHAR(30) NOT NULL,
    question_content TEXT NOT NULL,
    correct_answer  TEXT,
    student_answer  TEXT,
    error_type      VARCHAR(20) NOT NULL,
    source_type     VARCHAR(20) NOT NULL,
    source_id       BIGINT,
    difficulty      SMALLINT NOT NULL DEFAULT 1,
    review_count    INT NOT NULL DEFAULT 0,
    last_reviewed   TIMESTAMPTZ,
    is_resolved     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_eq_user ON error_questions(user_id);
CREATE INDEX idx_eq_subject ON error_questions(user_id, subject);
CREATE INDEX idx_eq_node ON error_questions(knowledge_node_id);
CREATE INDEX idx_eq_resolved ON error_questions(is_resolved);

-- 题库表
CREATE TABLE question_bank (
    id              BIGSERIAL PRIMARY KEY,
    subject         VARCHAR(30) NOT NULL,
    grade           SMALLINT NOT NULL,
    knowledge_node_id BIGINT,
    question_type   VARCHAR(30) NOT NULL,
    difficulty      SMALLINT NOT NULL DEFAULT 1,
    content         TEXT NOT NULL,
    options         JSONB,
    answer          TEXT NOT NULL,
    analysis        TEXT,
    source          VARCHAR(30) NOT NULL DEFAULT 'ai',
    exam_frequency  SMALLINT NOT NULL DEFAULT 1,
    is_valid        BOOLEAN NOT NULL DEFAULT TRUE,
    usage_count     INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_qb_subject_grade ON question_bank(subject, grade);
CREATE INDEX idx_qb_node ON question_bank(knowledge_node_id);
CREATE INDEX idx_qb_type_diff ON question_bank(question_type, difficulty);
CREATE INDEX idx_qb_exam_freq ON question_bank(exam_frequency);

-- 每日任务表
CREATE TABLE daily_tasks (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    task_date       DATE NOT NULL,
    subject         VARCHAR(30) NOT NULL DEFAULT '',
    task_mode       VARCHAR(10) NOT NULL,
    status          SMALLINT NOT NULL DEFAULT 1,
    total_items     INT NOT NULL DEFAULT 0,
    completed_items INT NOT NULL DEFAULT 0,
    correct_items   INT NOT NULL DEFAULT 0,
    time_limit_min  INT NOT NULL DEFAULT 0,
    actual_time_min INT,
    start_at        TIMESTAMPTZ,
    finish_at       TIMESTAMPTZ,
    pdf_url         VARCHAR(500),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_dt_user_date ON daily_tasks(user_id, task_date);
CREATE INDEX idx_dt_status ON daily_tasks(status);

-- 任务题目明细表
CREATE TABLE task_items (
    id              BIGSERIAL PRIMARY KEY,
    task_id         BIGINT NOT NULL REFERENCES daily_tasks(id) ON DELETE CASCADE,
    question_id     BIGINT,
    knowledge_node_id BIGINT REFERENCES knowledge_nodes(id),
    question_type   VARCHAR(30) NOT NULL,
    question_content TEXT NOT NULL,
    options         JSONB,
    correct_answer  TEXT NOT NULL,
    difficulty      SMALLINT NOT NULL DEFAULT 1,
    item_mode       VARCHAR(10) NOT NULL DEFAULT 'remedial',
    sort_order      INT NOT NULL DEFAULT 0,
    status          SMALLINT NOT NULL DEFAULT 1,
    student_answer  TEXT,
    is_correct      BOOLEAN,
    score           DECIMAL(5,2),
    answer_duration INT,
    grading_result  JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ti_task ON task_items(task_id);
CREATE INDEX idx_ti_status ON task_items(status);
CREATE INDEX idx_ti_node ON task_items(knowledge_node_id);

-- 用户已做题目记录
CREATE TABLE user_question_history (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    question_id     BIGINT NOT NULL REFERENCES question_bank(id),
    task_item_id    BIGINT REFERENCES task_items(id),
    is_correct      BOOLEAN,
    answered_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, question_id)
);
CREATE INDEX idx_uqh_user ON user_question_history(user_id);
CREATE INDEX idx_uqh_question ON user_question_history(question_id);

-- 风控记录表
CREATE TABLE risk_records (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    task_id         BIGINT REFERENCES daily_tasks(id),
    task_mode       VARCHAR(10) NOT NULL,
    violation_type  VARCHAR(30) NOT NULL,
    violation_level VARCHAR(10) NOT NULL,
    violation_count INT NOT NULL DEFAULT 1,
    detail          JSONB,
    handled         BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_rr_user ON risk_records(user_id);
CREATE INDEX idx_rr_task ON risk_records(task_id);
CREATE INDEX idx_rr_type ON risk_records(violation_type);

-- 学习诚信档案表
CREATE TABLE integrity_profiles (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT UNIQUE NOT NULL REFERENCES users(id),
    integrity_score DECIMAL(5,2) NOT NULL DEFAULT 100.00,
    total_violations INT NOT NULL DEFAULT 0,
    minor_count     INT NOT NULL DEFAULT 0,
    moderate_count  INT NOT NULL DEFAULT 0,
    severe_count    INT NOT NULL DEFAULT 0,
    tasks_invalidated INT NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ip_user ON integrity_profiles(user_id);

-- 学生能力层级表
CREATE TABLE student_ability_tiers (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    subject         VARCHAR(30) NOT NULL,
    tier            VARCHAR(20) NOT NULL,
    mastery_rate    DECIMAL(5,2) NOT NULL,
    advanced_ratio  INT NOT NULL DEFAULT 30,
    calculated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, subject)
);
CREATE INDEX idx_sat_user ON student_ability_tiers(user_id);

-- 学情报告表
CREATE TABLE learning_reports (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    report_type     VARCHAR(20) NOT NULL,
    period_start    DATE NOT NULL,
    period_end      DATE NOT NULL,
    overall_mastery DECIMAL(5,2),
    subject_summary JSONB NOT NULL,
    weak_points     JSONB,
    advanced_points JSONB,
    risk_summary    JSONB,
    ai_suggestions  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, report_type, period_start)
);
CREATE INDEX idx_lr_user ON learning_reports(user_id);
CREATE INDEX idx_lr_type ON learning_reports(report_type);

(in-package #:pgloader.transforms)

(defun sqlite-int-to-boolean (val)
  "SQLite stores booleans as INTEGER 0/1; PostgreSQL COPY expects boolean."
  (cond
    ((null val) :null)
    ((and (integerp val) (zerop val)) "false")
    ((and (integerp val) (= val 1)) "true")
    ((and (stringp val) (string= val "0")) "false")
    ((and (stringp val) (string= val "1")) "true")
    (t :null)))

(defun sqlite-text-to-jsonb (val)
  "SQLite JSON lives in TEXT; pass valid JSON through to PostgreSQL JSONB."
  (cond
    ((null val) :null)
    ((stringp val) val)
    (t (format nil "~a" val))))

(defun sqlite-timestamp-to-timestamp (val)
  "SQLite timestamps in TEXT/DATETIME columns; pass through for PostgreSQL TIMESTAMPTZ."
  (cond
    ((null val) :null)
    ((stringp val) val)
    (t (format nil "~a" val))))

(defun sqlite-text-to-uuid (val)
  "SQLite UUID keys live in TEXT; pass through for PostgreSQL UUID."
  (cond
    ((null val) :null)
    ((stringp val) val)
    (t (format nil "~a" val))))

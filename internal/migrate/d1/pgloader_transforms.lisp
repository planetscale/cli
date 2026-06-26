(in-package #:pgloader.transforms)

(defun sqlite-int-to-boolean (val)
  "SQLite stores booleans as INTEGER 0/1; PostgreSQL COPY expects boolean."
  (cond
    ((null val) :null)
    ((and (integerp val) (zerop val)) "false")
    ((and (stringp val) (string= val "0")) "false")
    (t "true")))

(defun sqlite-text-to-jsonb (val)
  "SQLite JSON lives in TEXT; pass valid JSON through to PostgreSQL JSONB."
  (cond
    ((null val) :null)
    ((stringp val) val)
    (t (format nil "~a" val))))

CREATE TABLE payment_method_countries (
    payment_method_code VARCHAR(50) NOT NULL REFERENCES payment_methods(code),
    country_code VARCHAR(2) NOT NULL REFERENCES countries(code),
    PRIMARY KEY (payment_method_code, country_code)
);

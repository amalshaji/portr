import re


def render_template(template, variables):
    pattern = r"{{\s*([^{}]*)\s*}}"

    def replace(match):
        var_name = match.group(1).strip()
        return str(variables.get(var_name, match.group(0)))

    rendered_template = re.sub(pattern, replace, template)
    return rendered_template

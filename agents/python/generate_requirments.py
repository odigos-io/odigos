import bootstrap_gen as b


def generate():
    print('Generating requirments.txt')
    with open('requirments.txt', 'a') as f:
        f.write('opentelemetry-distro[otlp]')
        f.write('\n')
        f.write('opentelemetry-instrumentation')
        f.write('\n')
        for l in b.libraries:
            f.write(b.libraries[l]['instrumentation'])
            f.write('\n')
        for d in b.default_instrumentations:
            f.write(d)
            f.write('\n')
    print('Done.')


generate()
